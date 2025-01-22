package main

import (
	"encoding/base64"
	"fmt"
	"os"
)

const (
	base64Table = "ABCDEFGHIJKLMNOPQRSTpqrstuvwxyz0123456789+/UVWXYZabcdefghijklmno"
)

var coder = base64.NewEncoding(base64Table)

func Base64Encode(src []byte) []byte {
	return []byte(coder.EncodeToString(src))
}

func Base64Decode(src []byte) ([]byte, error) {
	return coder.DecodeString(string(src))
}

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage: dcos_encrpt encode/decode [File Path]\n")
		return
	}
	operation := os.Args[1]
	inputFile := os.Args[2]
	if operation == "" || inputFile == "" {
		fmt.Printf("Usage: dcos_encrpt encode/decode [File Path]\n")
		return
	}

	buf, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "File Error: %s\n", err)
		return
	}
	if operation == "encode" {
		fmt.Printf("%s\n", string(Base64Encode(buf)))
	} else if operation == "decode" {
		result, errd := Base64Decode(buf)
		if errd != nil {
			fmt.Printf("File Error: %s\n", errd)
		} else {
			fmt.Printf("%s\n", string(result))
		}
	} else {
		fmt.Printf("Usage: dcos_encrpt encode/decode [File Path]\n")
	}
}
