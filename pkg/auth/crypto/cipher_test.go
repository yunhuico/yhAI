package crypto

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewASECipher(t *testing.T) {
	type args struct {
		key []byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test use empty key",
			args: args{
				key: nil,
			},
			wantErr: true,
		},
		{
			name: "test use 10 length key",
			args: args{
				key: bytes.Repeat([]byte{0x01}, 10),
			},
			wantErr: true,
		},
		{
			name: "test use 32 length key",
			args: args{
				key: bytes.Repeat([]byte{0x01}, 32),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewASECipher(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewASECipher() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestASECipher(t *testing.T) {
	c, err := NewASECipher([]byte("thisis32bitlongpassphraseimusing"))
	assert.NoError(t, err)

	t.Run("test encrypt empty", func(t *testing.T) {
		temp, err := c.Encrypt(nil)
		assert.NoError(t, err)

		src, err := c.Decrypt(temp)
		assert.NoError(t, err)
		assert.Len(t, src, 0)
	})

	t.Run("test decrypt empty", func(t *testing.T) {
		_, err := c.Decrypt(nil)
		assert.ErrorIs(t, ErrCiphertextTooShort, err)
	})

	t.Run("test decrypt length < 12 error", func(t *testing.T) {
		_, err := c.Decrypt(bytes.Repeat([]byte{0x01}, 11))
		assert.ErrorIs(t, ErrCiphertextTooShort, err)
	})

	t.Run("test encrypt json", func(t *testing.T) {
		jsonStr := `{
			"accessToken": "some-random-access-token",
		}`
		temp, err := c.Encrypt([]byte(jsonStr))
		assert.NoError(t, err)
		actual, err := c.Decrypt(temp)
		assert.NoError(t, err)
		assert.Equal(t, jsonStr, string(actual))
	})

	t.Run("test encrypt random data", func(t *testing.T) {
		for i := 0; i < 1; i++ {
			plaintext := randText(32)
			excepted, err := c.Encrypt(plaintext)
			assert.NoError(t, err)
			actual, err := c.Decrypt(excepted)
			assert.NoError(t, err)
			assert.Equal(t, plaintext, actual)
		}
	})

	type foo struct {
		Bar string `json:"bar"`
	}

	t.Run("test unmarshal error", func(t *testing.T) {
		var obj foo
		err := c.Unmarshal([]byte("{{"), &obj)
		assert.Error(t, err)
	})

	t.Run("test marshal and unmarshal success", func(t *testing.T) {
		obj := &foo{
			Bar: "some bar",
		}
		secret, err := c.Marshal(obj)
		assert.NoError(t, err)
		var obj2 foo
		err = c.Unmarshal(secret, &obj2)
		assert.NoError(t, err)
		assert.Equal(t, obj.Bar, obj2.Bar)
	})
}

var letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randText(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return b
}
