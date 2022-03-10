package crypto

import (
	"bytes"
	"crypto/subtle"
	"errors"
)

type PaddingMode interface {
	Padding([]byte, int) []byte
	UnPadding([]byte, int) ([]byte, error)
}

// PKCS5与PKCS7的区别：PKCS5用于块大小8Byte PKCS7用于块大小1-255Byte
// pkcs5作为pkcs7的子集算法，使用上没有什么区别，
// https://en.wikipedia.org/wiki/Padding_%28cryptography%29#PKCS#5_and_PKCS#7
// PKCS #5:https://www.ietf.org/rfc/rfc2898.txt
// PKCS #7:https://www.ietf.org/rfc/rfc2315.txt

var PKCS7Padding pkcs7Padding

type pkcs7Padding struct {
}

func (pkcs7Padding) Padding(plainTxt []byte, blockSize int) []byte {
	if blockSize < 1 || blockSize > 255 {
		return plainTxt
	}

	paddingBytes := blockSize - len(plainTxt)%blockSize
	return append(plainTxt, bytes.Repeat([]byte{byte(paddingBytes)}, paddingBytes)...)
}

func (pkcs7Padding) UnPadding(cipherTxt []byte, blockSize int) ([]byte, error) {
	if len(cipherTxt) == 0 {
		return nil, errors.New("cipherTxt is empty")
	}

	if len(cipherTxt)%blockSize != 0 {
		return nil, errors.New("cipherTxt not full block")
	}

	padLen := cipherTxt[len(cipherTxt)-1]
	toCheck := 255
	good := 1
	if toCheck > len(cipherTxt) {
		toCheck = len(cipherTxt)
	}
	for i := 0; i < toCheck; i++ {
		b := cipherTxt[len(cipherTxt)-1-i]

		outOfRange := subtle.ConstantTimeLessOrEq(int(padLen), i)
		equal := subtle.ConstantTimeByteEq(padLen, b)
		good &= subtle.ConstantTimeSelect(outOfRange, 1, equal)
	}

	good &= subtle.ConstantTimeLessOrEq(1, int(padLen))
	good &= subtle.ConstantTimeLessOrEq(int(padLen), len(cipherTxt))

	if good != 1 {
		return nil, errors.New("cipherTxt is not PKCS#7 padding")
	}

	return cipherTxt[:len(cipherTxt)-int(padLen)], nil
}

var ZeroPadding zeroPadding

type zeroPadding struct {
}

func (zeroPadding) Padding(plainTxt []byte, blockSize int) []byte {
	if blockSize < 1 || blockSize > 255 {
		return plainTxt
	}

	paddingBytes := blockSize - len(plainTxt)%blockSize
	return append(plainTxt, bytes.Repeat([]byte{byte(0)}, paddingBytes)...)
}

func (zeroPadding) UnPadding(cipherTxt []byte) ([]byte, error) {
	for i := len(cipherTxt) - 1; i >= 0; i-- {
		if cipherTxt[i] != 0 {
			cipherTxt = cipherTxt[:i+1]
			break
		}
	}

	return cipherTxt, nil
}
