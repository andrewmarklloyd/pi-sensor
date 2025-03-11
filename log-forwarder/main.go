package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	dataDogEndpoint = "https://http-intake.logs.datadoghq.com/api/v2/logs"
	maxRestarts     = 5
)

var hostname string

type syslog struct {
	Identifier string `json:"SYSLOG_IDENTIFIER"`
	Message    string `json:"MESSAGE"`
	Error      error
}

type response struct {
	Error string `json:"error"`
}

type ddBody struct {
	DDtags   string `json:"ddtags"`
	Hostname string `json:"hostname"`
	Message  string `json:"message"`
	Service  string `json:"service"`
	Source   string `json:"source"`
}

func main() {
	ddAPIKey := os.Getenv("DD_API_KEY")
	if ddAPIKey == "" {
		fmt.Println("DD_API_KEY env var must be set")
		os.Exit(1)
	}
	systemdUnits := os.Getenv("SYSTEMD_UNITS")
	if systemdUnits == "" {
		fmt.Println("SYSTEMD_UNITS env var must be set")
		os.Exit(1)
	}
	unitsSplit := strings.Split(systemdUnits, ",")
	units := []string{}
	for _, s := range unitsSplit {
		units = append(units, strings.Trim(s, " "))
	}

	for _, u := range units {
		go func() {
			monitorRestarts(u)
		}()
	}

	b, err := os.ReadFile("/etc/hostname")
	if err != nil {
		panic(err)
	}
	hostname = string(b)

	fmt.Println("Starting agent log forwarder")

	logChannel := make(chan syslog)
	go tailSystemdLogs(logChannel, units)
	for log := range logChannel {
		if log.Error != nil {
			fmt.Printf("error receiving logs from journalctl channel: %s\n", log.Error)
			break
		}

		err := sendLogs(log.Message, ddAPIKey)
		if err != nil {
			fmt.Println("error sending logs:", err)
		}
	}
}

func tailSystemdLogs(ch chan syslog, systemdUnits []string) {
	argsSlice := []string{}
	for _, s := range systemdUnits {
		argsSlice = append(argsSlice, "-u")
		argsSlice = append(argsSlice, s)
	}
	argsSlice = append(argsSlice, "-f", "-n 0", "--output", "json")
	cmd := exec.Command("journalctl", argsSlice...)
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("creating command stdout pipe: %s\n", err.Error())
		os.Exit(1)
	}

	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			var s syslog
			if err := json.Unmarshal([]byte(scanner.Text()), &s); err != nil {
				s.Error = fmt.Errorf("unmarshalling log: %s, original log text: %s", err, scanner.Text())
				ch <- s
				break
			}
			if s.Message != "" && s.Identifier != "systemd" && !strings.Contains(s.Message, "Logs begin at") {
				ch <- s
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		fmt.Printf("starting command: %s\n", err)
		os.Exit(1)
	}

	if err := cmd.Wait(); err != nil {
		close(ch)
		fmt.Printf("waiting for command: %s\n", err)
		os.Exit(1)
	}
}

func sendLogs(log, ddAPIKey string) error {
	b := ddBody{
		DDtags:   fmt.Sprintf("source:%s", hostname),
		Hostname: hostname,
		Message:  log,
		Service:  "pi-sensor",
		Source:   "pi-sensor-agent",
	}

	bodyBytes, err := json.Marshal(b)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", dataDogEndpoint, bytes.NewBuffer(bodyBytes))

	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Add("DD-API-KEY", ddAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var res response
	err = json.Unmarshal(body, &res)
	if err != nil {
		return err
	}

	if res.Error != "" {
		return fmt.Errorf(res.Error)
	}

	return nil
}

func monitorRestarts(systemdUnit string) {
	fmt.Println("Starting restart monitor")
	for range time.NewTicker(time.Minute * 5).C {
		cmd := exec.Command("systemctl", "show", systemdUnit, "-p", "NRestarts")

		stdout, err := cmd.Output()

		if err != nil {
			fmt.Printf("ERROR: %s\n", err.Error())
			continue
		}

		out := strings.TrimSuffix(string(stdout), "\n")
		split := strings.Split(out, "=")
		if len(split) != 2 {
			fmt.Printf("ERROR: expected output to be length 2 but got %d\n", len(split))
			continue
		}

		i, err := strconv.Atoi(split[1])
		if err != nil {
			fmt.Printf("ERROR: %s\n", err.Error())
		}

		if i > maxRestarts {
			fmt.Println("restart above limit, ensuring unit is stopped")
			cmd := exec.Command("sudo", "systemctl", "stop", systemdUnit)

			stdout, err := cmd.Output()
			if err != nil {
				fmt.Printf("ERROR: %s\n", err.Error())
				continue
			}

			fmt.Println(string(stdout))
		}
	}
}
