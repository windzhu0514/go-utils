package crypto

import (
	"crypto/aes"
	"crypto/des"
)

type DESCipher struct {
	key []byte
	cipherConfig
}

func NewDES(key []byte, opts ...Option) Cipher {
	c := &DESCipher{
		key: key,
	}

	for _, opt := range opts {
		opt(&c.cipherConfig)
	}

	if c.PaddingMode == nil {
		c.PaddingMode = PKCS7Padding
	}

	if c.BlockMode == nil {
		c.BlockMode = CBC
	}

	return c
}

func (c *DESCipher) Encrypt(plainTxt []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}

	return c.BlockMode.Encrypt(&c.cipherConfig, block, plainTxt)
}

func (c *DESCipher) Decrypt(cipherTxt []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}

	return c.BlockMode.Decrypt(&c.cipherConfig, block, cipherTxt)
}

type TripleDESCipher struct {
	key []byte
	cipherConfig
}

func NewTripleDES(key []byte, opts ...Option) Cipher {
	c := &TripleDESCipher{
		key: key,
	}

	for _, opt := range opts {
		opt(&c.cipherConfig)
	}

	if c.PaddingMode == nil {
		c.PaddingMode = PKCS7Padding
	}

	if c.BlockMode == nil {
		c.BlockMode = CBC
	}

	return c
}

func (c *TripleDESCipher) Encrypt(plainTxt []byte) ([]byte, error) {
	block, err := des.NewTripleDESCipher(c.key)
	if err != nil {
		return nil, err
	}

	return c.BlockMode.Encrypt(&c.cipherConfig, block, plainTxt)
}

func (c *TripleDESCipher) Decrypt(cipherTxt []byte) ([]byte, error) {
	block, err := des.NewTripleDESCipher(c.key)
	if err != nil {
		return nil, err
	}

	return c.BlockMode.Decrypt(&c.cipherConfig, block, cipherTxt)
}
