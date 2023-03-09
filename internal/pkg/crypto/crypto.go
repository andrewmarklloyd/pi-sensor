package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
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
