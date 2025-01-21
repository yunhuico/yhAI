package crypto_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth/crypto"
)

func FuzzAesCipherEncrypt(f *testing.F) {
	c, _ := crypto.NewASECipher(bytes.Repeat([]byte{0}, 32))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, err := c.Encrypt(data)
		assert.NoError(t, err)
	})
}

func FuzzAesCipherDecrypt(f *testing.F) {
	c, _ := crypto.NewASECipher(bytes.Repeat([]byte{0}, 32))

	f.Fuzz(func(t *testing.T, data []byte) {
		// ignore error, because we care about error, just make sure not panic
		_, _ = c.Decrypt(data)
	})
}
