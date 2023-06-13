package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"

	"github.com/windzhu0514/go-utils/crypto/ecb"
)

type cipherConfig struct {
	IV             []byte
	BlockMode      BlockMode
	PaddingMode    PaddingMode
	TagSize        int
	Nonce          []byte
	AdditionalData []byte
}

// AESCipher AES Advanced Encryption Standard
// 支持 ECB、CBC、CFB、CTR、GCM、OFB
// key 长度 16、24、32 位，即 128、192、256 bit（AES-128、AES-192、AES-256），位数越大安全性越高但加密速度越慢
type AESCipher struct {
	key []byte
	cipherConfig
}

type Option func(*cipherConfig)

func WithIV(iv []byte) Option {
	return func(a *cipherConfig) {
		a.IV = iv
	}
}

func WithBlockMode(bc BlockMode) Option {
	return func(a *cipherConfig) {
		a.BlockMode = bc
	}
}

func WithPaddingMode(padding PaddingMode) Option {
	return func(a *cipherConfig) {
		a.PaddingMode = padding
	}
}

func WithTagSize(tagSize int) Option {
	return func(a *cipherConfig) {
		a.TagSize = tagSize
	}
}

func WithNonce(nonce []byte) Option {
	return func(a *cipherConfig) {
		a.Nonce = nonce
	}
}

// gcm
func WithAdditionalData(additionalData []byte) Option {
	return func(a *cipherConfig) {
		a.AdditionalData = additionalData
	}
}

func NewAES(key []byte, opts ...Option) Cipher {
	c := &AESCipher{
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

func (c *AESCipher) Encrypt(plainTxt []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}

	return c.BlockMode.Encrypt(&c.cipherConfig, block, plainTxt)
}

func (c *AESCipher) Decrypt(cipherTxt []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}

	return c.BlockMode.Decrypt(&c.cipherConfig, block, cipherTxt)
}

// ECB 模式(lectronic codebook）
// 对每个明文块应用秘钥，缺点在于同样的平文块会被加密成相同的密文块，因此，它不能很好的隐藏数据模式
var ECB ecbBlockMode

type ecbBlockMode struct{}

func (e ecbBlockMode) Encrypt(c *cipherConfig, block cipher.Block, src []byte) ([]byte, error) {
	mode := ecb.NewECBEncrypter(block)
	src = c.PaddingMode.Padding(src, block.BlockSize())
	dst := make([]byte, len(src))
	mode.CryptBlocks(dst, src)

	return dst, nil
}

func (e ecbBlockMode) Decrypt(c *cipherConfig, block cipher.Block, src []byte) ([]byte, error) {
	mode := ecb.NewECBDecrypter(block)
	dst := make([]byte, len(src))
	mode.CryptBlocks(dst, src)

	return c.PaddingMode.UnPadding(dst, block.BlockSize())
}

// CBC 模式(Cipher-block chaining)
// 每个明文块先与前一个密文块进行异或后，再进行加密。在这种方法中，每个密文块都依赖于它前面的所有平文块。
// 同时，为了保证每条消息的唯一性，在第一个块中需要使用初始化向量。
// CBC是最为常用的工作模式。它的主要缺点在于加密过程是串行的，无法被并行化，而且消息必须被填充到块大小的整数倍。
// 加密时，平文中的微小改变会导致其后的全部密文块发生改变，而在解密时，从两个邻接的密文块中即可得到一个平文块。
// 因此，解密过程可以被并行化，而解密时，密文中一位的改变只会导致其对应的平文块完全改变和下一个平文块中对应位发生改变，
// 不会影响到其它平文的内容
var CBC cbc

type cbc struct{}

func (e cbc) Encrypt(c *cipherConfig, block cipher.Block, src []byte) ([]byte, error) {
	if len(c.IV) != block.BlockSize() {
		return nil, errors.New("IV length must equal block size")
	}

	mode := cipher.NewCBCEncrypter(block, c.IV)
	src = c.PaddingMode.Padding(src, block.BlockSize())
	dst := make([]byte, len(src))
	mode.CryptBlocks(dst, src)

	return dst, nil
}

func (e cbc) Decrypt(c *cipherConfig, block cipher.Block, src []byte) ([]byte, error) {
	if len(c.IV) != block.BlockSize() {
		return nil, errors.New("IV length must equal block size")
	}

	mode := cipher.NewCBCDecrypter(block, c.IV)
	dst := make([]byte, len(src))
	mode.CryptBlocks(dst, src)

	return c.PaddingMode.UnPadding(dst, block.BlockSize())
}

var CFB cfb

type cfb struct{}

func (e cfb) Encrypt(c *cipherConfig, block cipher.Block, src []byte) ([]byte, error) {
	if len(c.IV) != block.BlockSize() {
		return nil, errors.New("IV length must equal block size")
	}

	mode := cipher.NewCFBEncrypter(block, c.IV)
	dst := make([]byte, len(src))
	mode.XORKeyStream(dst, src)

	return dst, nil
}

func (e cfb) Decrypt(c *cipherConfig, block cipher.Block, src []byte) ([]byte, error) {
	if len(c.IV) != block.BlockSize() {
		return nil, errors.New("IV length must equal block size")
	}

	mode := cipher.NewCFBDecrypter(block, c.IV)
	dst := make([]byte, len(src))
	mode.XORKeyStream(dst, src)

	return dst, nil
}

var CTR ctr

type ctr struct{}

func (e ctr) Encrypt(c *cipherConfig, block cipher.Block, src []byte) ([]byte, error) {
	if len(c.IV) != block.BlockSize() {
		return nil, errors.New("IV length must equal block size")
	}

	mode := cipher.NewCTR(block, c.IV)
	dst := make([]byte, len(src))
	mode.XORKeyStream(dst, src)

	return dst, nil
}

func (e ctr) Decrypt(c *cipherConfig, block cipher.Block, src []byte) ([]byte, error) {
	if len(c.IV) != block.BlockSize() {
		return nil, errors.New("IV length must equal block size")
	}

	mode := cipher.NewCTR(block, c.IV)
	dst := make([]byte, len(src))
	mode.XORKeyStream(dst, src)

	return dst, nil
}

// GCM AE，Authenticated Encryption
var GCM gcm

type gcm struct{}

func (e gcm) Encrypt(c *cipherConfig, block cipher.Block, src []byte) ([]byte, error) {
	var (
		aesgcm cipher.AEAD
		err    error
	)

	if c.TagSize == 0 && len(c.Nonce) == 0 {
		aesgcm, err = cipher.NewGCM(block)
	} else if c.TagSize != 0 {
		aesgcm, err = cipher.NewGCMWithTagSize(block, c.TagSize)
	} else {
		aesgcm, err = cipher.NewGCMWithNonceSize(block, len(c.Nonce))
	}

	if err != nil {
		return nil, err
	}

	return aesgcm.Seal(nil, c.Nonce, src, c.AdditionalData), nil
}

func (e gcm) Decrypt(c *cipherConfig, block cipher.Block, src []byte) ([]byte, error) {
	var (
		aesgcm cipher.AEAD
		err    error
	)

	if c.TagSize == 0 && len(c.Nonce) == 0 {
		aesgcm, err = cipher.NewGCM(block)
	} else if c.TagSize != 0 {
		aesgcm, err = cipher.NewGCMWithTagSize(block, c.TagSize)
	} else {
		aesgcm, err = cipher.NewGCMWithNonceSize(block, len(c.Nonce))
	}

	if err != nil {
		return nil, err
	}

	return aesgcm.Open(nil, c.Nonce, src, c.AdditionalData)
}

var OFB ofb

type ofb struct{}

func (e ofb) Encrypt(c *cipherConfig, block cipher.Block, src []byte) ([]byte, error) {
	mode := cipher.NewOFB(block, c.IV)
	dst := make([]byte, len(src))
	mode.XORKeyStream(dst, src)

	return dst, nil
}

func (e ofb) Decrypt(c *cipherConfig, block cipher.Block, src []byte) ([]byte, error) {
	mode := cipher.NewOFB(block, c.IV)
	dst := make([]byte, len(src))
	mode.XORKeyStream(dst, src)

	return dst, nil
}
