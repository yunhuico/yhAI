package svc

import (
	"io/ioutil"
	"path/filepath"
	"testing"
)

// test helper
func readFile(t *testing.T, name string) []byte {
	bytes, err := ioutil.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}
