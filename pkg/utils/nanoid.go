package utils

import (
	"crypto/rand"
	"math/big"

	nanoID "github.com/matoous/go-nanoid/v2"
)

// nanoIDAlphabet URL and template safe
// consulting https://zelark.github.io/nano-id-cc/
// for collision rate.
// The alphabet can not contain lowercase due to support of Dkron.
// golang template variable name can't start with 0-9.
const nanoIDAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"

var lowerLetters = []byte("abcdefghijklmnopqrstuvwxyz")

// ShortNanoID size = 9, ~60 days for 1% collision rate
func ShortNanoID() (id string, err error) {
	return genNanoID(9)
}

// NanoID size = 16
// 1000 IDs per hour,
// ~46 thousand years needed,
// in order to have a 1% probability of at least one collision.
func NanoID() (id string, err error) {
	return genNanoID(16)
}

// LongNanoID size = 21
// 1000 IDs per second,
// ~355 million years needed, in order to have a 1% probability of at least one collision.
func LongNanoID() (string, error) {
	return genNanoID(21)
}

// genNanoID ensures the length>1
func genNanoID(length int) (id string, err error) {
	id, err = nanoID.Generate(nanoIDAlphabet, length-1)
	if err != nil {
		return
	}
	id = genFirstLetter() + id
	return
}

// genFirstLetter ensures the first letter not contains digits.
// golang template variable name can't start with 0-9.
func genFirstLetter() string {
	startAlphabet := byte('a')
	randIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(lowerLetters))))
	if err == nil {
		startAlphabet = lowerLetters[randIndex.Int64()]
	}
	return string(startAlphabet)
}
