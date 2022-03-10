package crypto

import (
	"crypto/aes"
	"crypto/cipher"

	"github.com/windzhu0514/go-utils/crypto/ecb"
)

// AES Advanced Encryption Standard
// https://blog.csdn.net/xiaohu50/article/details/51682849
// ecb、cbc、ofb、cfb
// AES 对加密 key 的长度要求必须固定为 16、24、32 位，也就是 128、192、256 比特，
// 所以又有一个 AES-128、AES-192、AES-256 这种叫法，位数越大安全性越高但加密速度越慢。
// 最关键是对明文长度也有要求，必须是分组长度长度的倍数，AES 加密数据块分组长度必须为
// 128bit 也就是 16 位，所以这块又涉及到一个填充问题，而这个填充方式可以分为 PKCS7 和 PKCS5 等方式，

// ECB和CBC需要填充

type AESCipher struct {
	Key            []byte // 对称加密Key或者公钥私钥
	IV             []byte
	BlockMode      BlockMode
	PaddingMode    PaddingMode
	TagSize        int
	Nonce          []byte
	AdditionalData []byte
}

type Option func(*AESCipher)

func WithIV(iv []byte) Option {
	return func(a *AESCipher) {
		a.IV = iv
	}
}

func WithBlockMode(bc BlockMode) Option {
	return func(a *AESCipher) {
		a.BlockMode = bc
	}
}

func WithPaddingMode(padding PaddingMode) Option {
	return func(a *AESCipher) {
		a.PaddingMode = padding
	}
}

func WithTagSize(tagSize int) Option {
	return func(a *AESCipher) {
		a.TagSize = tagSize
	}
}

func WithNonce(nonce []byte) Option {
	return func(a *AESCipher) {
		a.Nonce = nonce
	}
}

func WithAdditionalData(additionalData []byte) Option {
	return func(a *AESCipher) {
		a.AdditionalData = additionalData
	}
}

func NewAES(key []byte, opts ...Option) *AESCipher {
	c := &AESCipher{
		Key: key,
	}

	for _, opt := range opts {
		opt(c)
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
	block, err := aes.NewCipher(c.Key)
	if err != nil {
		return nil, err
	}

	return c.BlockMode.Encrypt(c, block, plainTxt)
}

func (c *AESCipher) Decrypt(cipherTxt []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.Key)
	if err != nil {
		return nil, err
	}

	return c.BlockMode.Decrypt(c, block, cipherTxt)
}

type BlockMode interface {
	Encrypt(*AESCipher, cipher.Block, []byte) ([]byte, error)
	Decrypt(*AESCipher, cipher.Block, []byte) ([]byte, error)
}

// ECB 模式(lectronic codebook）
// 对每个明文块应用秘钥，缺点在于同样的平文块会被加密成相同的密文块，因此，它不能很好的隐藏数据模式
var ECB ecbBlockMode

type ecbBlockMode struct {
}

func (e ecbBlockMode) Encrypt(c *AESCipher, block cipher.Block, src []byte) ([]byte, error) {
	mode := ecb.NewECBEncrypter(block)
	src = c.PaddingMode.Padding(src, block.BlockSize())
	dst := make([]byte, len(src))
	mode.CryptBlocks(dst, src)

	return dst, nil
}

func (e ecbBlockMode) Decrypt(c *AESCipher, block cipher.Block, src []byte) ([]byte, error) {
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

type cbc struct {
}

func (e cbc) Encrypt(c *AESCipher, block cipher.Block, src []byte) ([]byte, error) {
	mode := cipher.NewCBCEncrypter(block, c.IV)
	src = c.PaddingMode.Padding(src, block.BlockSize())
	dst := make([]byte, len(src))
	mode.CryptBlocks(dst, src)

	return dst, nil
}

func (e cbc) Decrypt(c *AESCipher, block cipher.Block, src []byte) ([]byte, error) {
	mode := cipher.NewCBCDecrypter(block, c.IV)
	dst := make([]byte, len(src))
	mode.CryptBlocks(dst, src)

	return c.PaddingMode.UnPadding(dst, block.BlockSize())
}

var CFB cfb

type cfb struct {
}

func (e cfb) Encrypt(c *AESCipher, block cipher.Block, src []byte) ([]byte, error) {
	mode := cipher.NewCFBEncrypter(block, c.IV)
	dst := make([]byte, len(src))
	mode.XORKeyStream(dst, src)

	return dst, nil
}

func (e cfb) Decrypt(c *AESCipher, block cipher.Block, src []byte) ([]byte, error) {
	mode := cipher.NewCFBDecrypter(block, c.IV)
	dst := make([]byte, len(src))
	mode.XORKeyStream(dst, src)

	return dst, nil
}

var CTR ctr

type ctr struct {
}

func (e ctr) Encrypt(c *AESCipher, block cipher.Block, src []byte) ([]byte, error) {
	mode := cipher.NewCTR(block, c.IV)
	dst := make([]byte, len(src))
	mode.XORKeyStream(dst, src)

	return dst, nil
}

func (e ctr) Decrypt(c *AESCipher, block cipher.Block, src []byte) ([]byte, error) {
	mode := cipher.NewCTR(block, c.IV)
	dst := make([]byte, len(src))
	mode.XORKeyStream(dst, src)

	return dst, nil
}

// GCM AE，Authenticated Encryption
var GCM gcm

type gcm struct {
}

func (e gcm) Encrypt(c *AESCipher, block cipher.Block, src []byte) ([]byte, error) {
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

func (e gcm) Decrypt(c *AESCipher, block cipher.Block, src []byte) ([]byte, error) {
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

type ofb struct {
}

func (e ofb) Encrypt(c *AESCipher, block cipher.Block, src []byte) ([]byte, error) {
	mode := cipher.NewOFB(block, c.IV)
	dst := make([]byte, len(src))
	mode.XORKeyStream(dst, src)

	return dst, nil
}

func (e ofb) Decrypt(c *AESCipher, block cipher.Block, src []byte) ([]byte, error) {
	mode := cipher.NewOFB(block, c.IV)
	dst := make([]byte, len(src))
	mode.XORKeyStream(dst, src)

	return dst, nil
}
