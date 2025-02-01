package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
)

type Util struct {
	gcm cipher.AEAD
}

func NewUtil(key string) (Util, error) {
	c, err := aes.NewCipher([]byte(key))
	if err != nil {
		return Util{}, fmt.Errorf("creating new aes cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return Util{}, fmt.Errorf("creating new gcm: %w", err)
	}

	return Util{
		gcm: gcm,
	}, nil
}

func (u *Util) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, u.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("reading nonce: %w", err)
	}

	return u.gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (u *Util) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := u.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return u.gcm.Open(nil, nonce, ciphertext, nil)
}

func GetCertTokenMetadataExp(host, port string) (config.TokenMetadata, error) {
	dialer := &net.Dialer{}
	conn, err := tls.DialWithDialer(
		dialer,
		"tcp",
		host+":"+port,
		&tls.Config{
			InsecureSkipVerify: false,
		})
	if err != nil {
		return config.TokenMetadata{}, err
	}

	defer conn.Close()

	if err := conn.Handshake(); err != nil {
		return config.TokenMetadata{}, err
	}

	pc := conn.ConnectionState().PeerCertificates
	certs := make([]*x509.Certificate, 0, len(pc))
	for _, cert := range pc {
		if cert.IsCA {
			continue
		}
		certs = append(certs, cert)
	}

	if len(certs) != 1 {
		return config.TokenMetadata{}, fmt.Errorf("expected number of certs to be 1 but was %d", len(certs))
	}

	if len(certs[0].DNSNames) != 1 {
		return config.TokenMetadata{}, fmt.Errorf("expected number of dns names to be 1 but was %d", len(certs[0].DNSNames))
	}

	tm := config.TokenMetadata{
		Name:       host,
		Owner:      "server",
		Expiration: certs[0].NotAfter.Format(time.RFC3339),
	}
	return tm, nil
}
