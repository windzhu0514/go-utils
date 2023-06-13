package crypto

import (
	"crypto/aes"
	"crypto/des"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"testing"
)

func TestAESCBC(t *testing.T) {
	key := []byte("1q2w3e4r5t6y7u8i")
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	aesCipher := NewAES(key, WithIV(iv), WithPaddingMode(PKCS7Padding))
	cipherTxt, err := aesCipher.Encrypt([]byte("fsdf"))
	base64Txt := base64.StdEncoding.EncodeToString(cipherTxt)
	fmt.Println(base64Txt, err)

	cipherTxt, _ = base64.StdEncoding.DecodeString(base64Txt)
	plainTxt, err := aesCipher.Decrypt(cipherTxt)
	fmt.Println(string(plainTxt), err)
}

func TestTripleDESCBC(t *testing.T) {
	key := []byte("1q2w3e4r5t6y7u8i9o0p1q2w")
	iv := make([]byte, des.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	ci := NewTripleDES(key, WithIV(iv), WithPaddingMode(PKCS7Padding), WithBlockMode(CBC))
	cipherTxt, err := ci.Encrypt([]byte("fsdf"))
	base64Txt := base64.StdEncoding.EncodeToString(cipherTxt)
	fmt.Println(base64Txt, err)

	cipherTxt, _ = base64.StdEncoding.DecodeString(base64Txt)
	plainTxt, err := ci.Decrypt(cipherTxt)
	fmt.Println(string(plainTxt), err)
}
