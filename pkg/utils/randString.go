package utils

import (
	"crypto/rand"
	"math/big"
)

var letters = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

func RandStr(length int) (string, error) {
	var result []byte
	for i := 0; i < length; i++ {
		randIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		result = append(result, letters[randIndex.Int64()])
	}
	return string(result), nil
}

// UnsafeRandStr just using in testing.
func UnsafeRandStr(length int) string {
	str, _ := RandStr(length)
	return str
}
