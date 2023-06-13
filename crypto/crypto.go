package crypto

import "crypto/cipher"

type Cipher interface {
	Encrypt(plainTxt []byte) ([]byte, error)
	Decrypt(cipherTxt []byte) ([]byte, error)
}

type BlockMode interface {
	Encrypt(*cipherConfig, cipher.Block, []byte) ([]byte, error)
	Decrypt(*cipherConfig, cipher.Block, []byte) ([]byte, error)
}
