package common

import (
	"bytes"
	"crypto/cipher"
	"errors"
	"encoding/base64"
	//	"errors"
	//	"github.com/Sirupsen/logrus"
	"github.com/magiconair/properties"
	"crypto/des"
	//	"github.com/samuel/go-zookeeper/zk"
	//	"math/rand"
	//	"strings"
	//	"time"
)

var UTIL *Util
var (
	clusterMgmtPath      string = "/cluster"
	clusterMgmtEndpoints []string
)

type Util struct {
	Props *properties.Properties
}

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

func DesEncrypt(origData, key []byte) ([]byte, error) {
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}
	origData = pkcs5Padding(origData, block.BlockSize())

	blockMode := cipher.NewCBCEncrypter(block, key)
	crypted := make([]byte, len(origData))
	if len(origData)%blockMode.BlockSize() != 0 {
		return nil, errors.New("failed to encrypt due to invalid encrypt message")
	}
	blockMode.CryptBlocks(crypted, origData)
	return crypted, nil
}

func DesDecrypt(crypted, key []byte) ([]byte, error) {
	if len(key) != 8 {
        	return nil, errors.New("key length must be 8 bytes")
    	}
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockMode := cipher.NewCBCDecrypter(block, key)
	origData := make([]byte, len(crypted))
	if len(crypted)%blockMode.BlockSize() != 0 {
		return nil, errors.New("failed to decrypt due to invalid decrypt message")
	}
	blockMode.CryptBlocks(origData, crypted)
	origData = pkcs5UnPadding(origData)
	return origData, nil
}

func pkcs5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func pkcs5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

