package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"text/template"

	"github.com/avast/retry-go/v5"

	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/digitalocean/godo"
)

const certbotCertCmdTemplate = `
set -e

sudo certbot certonly \
    --non-interactive \
    --agree-tos \
    --manual \
    -m {{.Email}} \
    --preferred-challenge http \
    --manual-auth-hook /home/mqtt-server/certbot/authenticator.sh \
    -d {{.MosquittoDomain}}{{if .DryRun}} \
    --dry-run{{else}}{{end}}`

var client *godo.Client

func main() {

	command := os.Getenv("COMMAND")
	if command == "" {
		log.Fatalln("COMMAND env var must be set")
	}

	firewallName := os.Getenv("FIREWALL_NAME")
	if firewallName == "" {
		log.Fatalln("FIREWALL_NAME env var must be set")
	}

	sshUserHost := os.Getenv("SSH_USER_HOST")
	if sshUserHost == "" {
		log.Fatalln("SSH_USER_HOST env var must be set")
	}

	inboundSourceAddr := os.Getenv("INBOUND_SOURCE_ADDR")
	if inboundSourceAddr == "" {
		log.Fatalln("INBOUND_SOURCE_ADDR env var must be set")
	}

	email := os.Getenv("EMAIL_ADDR")
	if email == "" {
		log.Fatalln("EMAIL_ADDR env var must be set")
	}

	mosquittoDomain := os.Getenv("MOSQUITTO_DOMAIN")
	if mosquittoDomain == "" {
		log.Fatalln("MOSQUITTO_DOMAIN env var must be set")
	}

	dryRunStr := os.Getenv("DRY_RUN")
	if dryRunStr == "" {
		log.Fatalln("DRY_RUN env var must be set")
	}
	dryRun := dryRunStr == "true"

	doAccessToken := os.Getenv("DO_ACCESS_TOKEN")
	if doAccessToken == "" {
		log.Fatalln("DO_ACCESS_TOKEN env var must be set")
	}

	ctx := context.Background()
	client = godo.NewFromToken(doAccessToken)

	var err error
	switch command {
	case "renew":
		err = renewCert(ctx, firewallName, sshUserHost, inboundSourceAddr, email, mosquittoDomain, dryRun)
	case "cleanup":
		err = cleanup(ctx, firewallName, sshUserHost)
	default:
		log.Fatalln("COMMAND env var must be set to 'renew' or 'cleanup'")
	}

	if err != nil {
		log.Fatalf("error running command %s: %s\n", command, err.Error())
	}
	fmt.Printf("successfully ran command %s\n", command)
}

func renewCert(ctx context.Context, firewallName, sshUserHost, inboundSourceAddr, email, mosquittoDomain string, dryRun bool) error {
	f, err := getFirewall(ctx, client, firewallName)
	if err != nil {
		return err
	}

	if f.ID == "" {
		fmt.Println("firewall not found, creating it now")
		err = createFirewall(ctx, firewallName, inboundSourceAddr)
		if err != nil {
			return err
		}
	} else {
		fmt.Println("firewall already exists")
	}

	fmt.Println("building cert server")
	if err := buildCertServer(); err != nil {
		return err
	}

	fmt.Println("ensuring cert server stopped")
	if err := ensureCertServerStopped(sshUserHost); err != nil {
		return err
	}

	fmt.Println("deploying assets")
	if err := deployAssets(sshUserHost); err != nil {
		return err
	}

	fmt.Println("starting server")
	if err := startServer(sshUserHost); err != nil {
		return err
	}

	fmt.Println("running cerbot cert creation")
	if err := runCerbotCertRequest(sshUserHost, email, mosquittoDomain, dryRun); err != nil {
		return err
	}

	fmt.Println("installing certs and restarting mosquitto")
	if err := installCertsAndRestart(sshUserHost, mosquittoDomain); err != nil {
		return err
	}

	fmt.Println("cleanup up resources")
	if err := cleanup(ctx, firewallName, sshUserHost); err != nil {
		return fmt.Errorf("cleaning up")
	}

	return nil
}

func cleanup(ctx context.Context, firewallName, sshUserHost string) error {
	f, err := getFirewall(ctx, client, firewallName)
	if err != nil {
		return err
	}

	if f.ID == "" {
		fmt.Println("firewall already deleted")
	} else {
		fmt.Println("deleting firewall")
		_, err = client.Firewalls.Delete(ctx, f.ID)
		if err != nil {
			return err
		}
	}

	fmt.Println("ensuring cert server stopped")
	if err := ensureCertServerStopped(sshUserHost); err != nil {
		return err
	}

	return nil
}

func getFirewall(ctx context.Context, c *godo.Client, firewallName string) (godo.Firewall, error) {
	firewalls, _, err := c.Firewalls.List(ctx, &godo.ListOptions{})
	if err != nil {
		return godo.Firewall{}, fmt.Errorf("listing firewalls: %s", err)
	}

	for _, f := range firewalls {
		if f.Name == firewallName {
			return f, nil
		}
	}

	return godo.Firewall{}, nil
}

func createFirewall(ctx context.Context, firewallName, inboundSourceAddr string) error {
	_, _, err := client.Firewalls.Create(ctx, &godo.FirewallRequest{
		Name: firewallName,
		Tags: []string{"mqtt-server"},
		InboundRules: []godo.InboundRule{
			{
				Protocol:  "tcp",
				PortRange: "80",
				Sources: &godo.Sources{
					Addresses: []string{
						inboundSourceAddr,
					},
				},
			},
		},
		OutboundRules: []godo.OutboundRule{
			{
				Protocol:  "tcp",
				PortRange: "all",
				Destinations: &godo.Destinations{
					Addresses: []string{"0.0.0.0/0"},
				},
			},
		},
	})

	return err
}

func buildCertServer() error {
	cmd := exec.Command(
		"bash", "-c",
		`
set -e
GOOS=linux CGO_ENABLED=0 GOARCH=amd64 go build -o internal/cmd/update-cert/assets/cert-server internal/cmd/update-cert/server/main.go
`,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("building cert server failed: %v\nstderr:\n%s\nstdout:\n%s", err, stderr.String(), stdout.String())
	}

	return nil
}

func deployAssets(sshUserHost string) error {
	files, err := filepath.Glob("internal/cmd/update-cert/assets/*")
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return errors.New("no asset files matched glob")
	}

	dest := fmt.Sprintf("%s:/home/mqtt-server/certbot", sshUserHost)
	args := append(files, dest)

	cmd := exec.Command("scp", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("scp failed: %v\nstderr:\n%s\nstdout:\n%s", err, stderr.String(), stdout.String())
	}

	return nil
}

func startServer(sshUserHost string) error {
	cmd := exec.Command(
		"ssh",
		sshUserHost,
		"bash", "-c",
		`
set -e
sudo rm -f /tmp/shutdown
cd /home/mqtt-server/certbot
sudo nohup ./cert-server > /tmp/server.log 2>&1 &
`,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("starting server failed: %v\nstderr:\n%s\nstdout:\n%s", err, stderr.String(), stdout.String())
	}

	return nil
}

func ensureCertServerStopped(sshUserHost string) error {
	cmd := exec.Command(
		"ssh",
		sshUserHost,
		"bash", "-c",
		`
set -e
touch /tmp/shutdown
sleep 2
`,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("starting server failed: %v\nstderr:\n%s\nstdout:\n%s", err, stderr.String(), stdout.String())
	}

	return nil
}

func runCerbotCertRequest(sshUserHost, email, mosquittoDomain string, dryRun bool) error {
	err := retry.New(
		retry.Attempts(5),
		retry.Delay(time.Second),
	).Do(
		func() error {
			resp, err := http.Get(fmt.Sprintf("http://%s", mosquittoDomain))
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("expected status ok but got %d", resp.StatusCode)
			}
			return nil
		},
	)

	if err != nil {
		return err
	}

	tmpl := template.Must(template.New("certrenew").Parse(certbotCertCmdTemplate))

	data := struct {
		Email           string
		MosquittoDomain string
		DryRun          bool
	}{email, mosquittoDomain, dryRun}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return err
	}

	result := buf.String()

	cmd := exec.Command(
		"ssh",
		sshUserHost,
		"bash", "-c",
		result,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("starting server failed: %v\nstderr:\n%s\nstdout:\n%s", err, stderr.String(), stdout.String())
	}

	fmt.Println(stdout.String())

	return nil
}

func installCertsAndRestart(sshUserHost, mosquittoDomain string) error {
	script := `set -euo pipefail
cd /etc/mosquitto/certs/%s/
for t in cert chain fullchain privkey; do
	# find the newest cert version by last index of files
	src=$(ls /etc/letsencrypt/archive/%s/${t}*.pem | sort -V | tail -n 1)
	cp -f "$src" "${t}.pem"
done

chmod 644 cert*.pem chain*.pem fullchain*.pem
chown root:mosquitto privkey*.pem
chmod 640 privkey*.pem

echo "new expiration:"
openssl x509 -in cert.pem -noout -enddate

systemctl restart mosquitto
`

	b64 := base64.StdEncoding.EncodeToString(fmt.Appendf(nil, script, mosquittoDomain, mosquittoDomain))
	remote := fmt.Sprintf(`sudo -n bash -lc "$(echo %s | base64 -d)"`, b64)

	cmd := exec.Command("ssh", sshUserHost, remote)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("installing certs failed: %v\nstderr:\n%s\nstdout:\n%s", err, stderr.String(), stdout.String())
	}

	fmt.Println(stdout.String())

	return nil
}
