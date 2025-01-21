package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"

	"github.com/pkg/errors"
)

//go:generate mockgen -source=$GOFILE -destination=./mock/mock_$GOFILE -package=cryptoMock
type (
	// CryptoCipher basic crypto algorithm
	CryptoCipher interface {
		Encrypt([]byte) ([]byte, error)
		Decrypt([]byte) ([]byte, error)
		ObjectMarshaler
	}

	// ObjectMarshaler interface for marshaling object by cipher
	ObjectMarshaler interface {
		Marshal(any) ([]byte, error)
		Unmarshal([]byte, any) error
	}

	// aesCipher is a symmetric algorithm
	aesCipher struct {
		// call aes in gcm mode
		// reference: https://en.wikipedia.org/wiki/Galois/Counter_Mode
		gcm cipher.AEAD
	}
)

func (a *aesCipher) Marshal(obj any) (dist []byte, err error) {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		err = fmt.Errorf("failed to marshal object in aes cipher: %w", err)
		return
	}
	return a.Encrypt(jsonBytes)
}

func (a *aesCipher) Unmarshal(plaintext []byte, obj any) error {
	jsonBytes, err := a.Decrypt(plaintext)
	if err != nil {
		return fmt.Errorf("failed to decrypt plaintext in aes cipher: %w", err)
	}
	return json.Unmarshal(jsonBytes, obj)
}

// NewASECipher will create a new AES cipher
func NewASECipher(key []byte) (CryptoCipher, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	return &aesCipher{gcm: gcm}, nil
}

func (a *aesCipher) Encrypt(plaintext []byte) (dst []byte, err error) {
	nonce := make([]byte, a.gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return a.gcm.Seal(nonce, nonce, plaintext, nil), nil
}

var ErrCiphertextTooShort = errors.New("ciphertext too short")

func (a *aesCipher) Decrypt(ciphertext []byte) (dst []byte, err error) {
	nonceSize := a.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrCiphertextTooShort
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return a.gcm.Open(nil, nonce, ciphertext, nil)
}
