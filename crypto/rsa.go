package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"hash"
	"strings"
)

type RSACipher struct {
	pubKey []byte
	pub    *rsa.PublicKey
	priKey []byte
	priv   *rsa.PrivateKey
}

type RSACipherOption func(*RSACipher)

func NewRSA(opts ...RSACipherOption) (*RSACipher, error) {
	rsa := &RSACipher{}
	for _, opt := range opts {
		opt(rsa)
	}

	if rsa.pubKey == nil && rsa.priKey == nil {
		return nil, errors.New("both public key and private key are empty")
	}

	if rsa.pubKey != nil {
		var err error
		rsa.pub, err = ParseRSAPublicKey(rsa.pubKey)
		if err != nil {
			return nil, err
		}
	}

	if rsa.priKey != nil {
		var err error
		rsa.priv, err = ParseRSAPrivateKey(rsa.priKey)
		if err != nil {
			return nil, err
		}
	}

	return rsa, nil
}

func WithPublicKey(pubKey []byte) RSACipherOption {
	return func(r *RSACipher) {
		r.pubKey = pubKey
	}
}

func WithPrivateKey(priKey []byte) RSACipherOption {
	return func(r *RSACipher) {
		r.priKey = priKey
	}
}

func (r *RSACipher) EncryptPKCS1v15(plainText []byte) ([]byte, error) {
	if r.pub == nil {
		return nil, errors.New("public key is empty")
	}

	cipherText, err := rsa.EncryptPKCS1v15(rand.Reader, r.pub, plainText)
	if err != nil {
		return nil, err
	}

	return cipherText, nil
}

func (r *RSACipher) EncryptPKCS1v15Hex(plainText []byte) ([]byte, error) {
	cipherText, err := r.EncryptPKCS1v15(plainText)
	if err != nil {
		return nil, err
	}

	hexBytes := make([]byte, hex.EncodedLen(len(cipherText)))
	hex.Encode(hexBytes, cipherText)

	return hexBytes, nil
}

func (r *RSACipher) EncryptPKCS1v15Base64(plainText []byte) ([]byte, error) {
	cipherText, err := r.EncryptPKCS1v15(plainText)
	if err != nil {
		return nil, err
	}

	base64Bytes := make([]byte, base64.StdEncoding.EncodedLen(len(cipherText)))
	base64.StdEncoding.Encode(base64Bytes, cipherText)

	return base64Bytes, nil
}

func (r *RSACipher) EncryptOAEP(hash hash.Hash, plainText []byte, label []byte) ([]byte, error) {
	if r.pub == nil {
		return nil, errors.New("public key is empty")
	}

	if hash == nil {
		hash = sha256.New()
	}

	cipherText, err := rsa.EncryptOAEP(hash, rand.Reader, r.pub, plainText, label)
	if err != nil {
		return nil, err
	}

	return cipherText, nil
}

func (r *RSACipher) EncryptOAEPHex(hash hash.Hash, plainText []byte, label []byte) ([]byte, error) {
	cipherText, err := r.EncryptOAEP(hash, plainText, label)
	if err != nil {
		return nil, err
	}

	hexBytes := make([]byte, hex.EncodedLen(len(cipherText)))
	hex.Encode(hexBytes, cipherText)

	return hexBytes, nil
}

func (r *RSACipher) EncryptOAEPBase64(hash hash.Hash, plainText []byte, label []byte) ([]byte, error) {
	cipherText, err := r.EncryptOAEP(hash, plainText, label)
	if err != nil {
		return nil, err
	}

	base64Bytes := make([]byte, base64.StdEncoding.EncodedLen(len(cipherText)))
	base64.StdEncoding.Encode(base64Bytes, cipherText)

	return base64Bytes, nil
}

func (r *RSACipher) DecryptPKCS1v15(cipherText []byte) ([]byte, error) {
	if r.priv == nil {
		return nil, errors.New("private key is empty")
	}

	plainText, err := rsa.DecryptPKCS1v15(rand.Reader, r.priv, cipherText)
	if err != nil {
		return nil, err
	}

	return plainText, nil
}

func (r *RSACipher) DecryptPKCS1v15Hex(cipherText []byte) ([]byte, error) {
	cipherBytes := make([]byte, hex.DecodedLen(len(cipherText)))
	n, err := hex.Decode(cipherBytes, cipherText)
	if err != nil {
		return nil, err
	}

	return r.DecryptPKCS1v15(cipherBytes[:n])
}

func (r *RSACipher) DecryptPKCS1v15Base64(cipherText []byte) ([]byte, error) {
	cipherBytes := make([]byte, base64.StdEncoding.DecodedLen(len(cipherText)))
	n, err := base64.StdEncoding.Decode(cipherBytes, cipherText)
	if err != nil {
		return nil, err
	}

	return r.DecryptPKCS1v15(cipherBytes[:n])
}

func (r *RSACipher) DecryptOAEP(hash hash.Hash, cipherText []byte, label []byte) ([]byte, error) {
	if r.priv == nil {
		return nil, errors.New("private key is empty")
	}

	if hash == nil {
		hash = sha256.New()
	}

	plainText, err := rsa.DecryptOAEP(hash, rand.Reader, r.priv, cipherText, label)
	if err != nil {
		return nil, err
	}

	return plainText, nil
}

func (r *RSACipher) DecryptOAEPHex(hash hash.Hash, cipherText []byte, label []byte) ([]byte, error) {
	cipherBytes := make([]byte, hex.DecodedLen(len(cipherText)))
	n, err := hex.Decode(cipherBytes, cipherText)
	if err != nil {
		return nil, err
	}

	return r.DecryptOAEP(hash, cipherBytes[:n], label)
}

func (r *RSACipher) DecryptOAEPBase64(hash hash.Hash, cipherText []byte, label []byte) ([]byte, error) {
	cipherBytes := make([]byte, base64.StdEncoding.DecodedLen(len(cipherText)))
	n, err := base64.StdEncoding.Decode(cipherBytes, cipherText)
	if err != nil {
		return nil, err
	}

	return r.DecryptOAEP(hash, cipherBytes[:n], label)
}

// ParseRSAPublicKey 从 ASN.1 PEM 格式的证书中解析 PKCS #1 格式或者 PKIX 格式的 RSA 公钥
func ParseRSAPublicKey(certPEM []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, errors.New("decode PEM data failed")
	}

	pubKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		if strings.Contains(err.Error(), "use ParsePKIXPublicKey instead for this key format") {
			pub, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				return nil, err
			}

			pubKey, ok := pub.(*rsa.PublicKey)
			if !ok {
				return nil, errors.New("pem data is not a valid RSA public key")
			}

			return pubKey, nil
		}

		return nil, err
	}

	return pubKey, nil
}

// ParseRSAPKCS1PublicKey 从 ASN.1 PEM 格式的证书中解析 PKCS #1 格式的 RSA 公钥，PEM 块类型通常是 "RSA PUBLIC KEY"
func ParseRSAPKCS1PublicKey(certPEM []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "RSA PUBLIC KEY" {
		return nil, errors.New("decode PEM data failed")
	}

	return x509.ParsePKCS1PublicKey(block.Bytes)
}

// ParseRSAPKIXPublicKey 从 ASN.1 PEM 格式的证书中解析中解析 PKIX 格式的 RSA 公钥，PEM 块类型通常是 "PUBLIC KEY"
func ParseRSAPKIXPublicKey(certPEM []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("decode PEM data failed")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("pem data is not a valid RSA public key")
	}

	return rsaPub, nil
}

// ParsePKIXPublicKey 从 ASN.1 PEM 格式的证书中解析 PKIX 格式的公钥，PEM 块类型通常是 "PUBLIC KEY"
func ParsePKIXPublicKey(certPEM []byte) (any, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("decode PEM data failed")
	}

	return x509.ParsePKIXPublicKey(block.Bytes)
}

// ParseRSAPublicKeyFromCert 从 ASN.1 PEM 格式的证书中解析 X.509 证书，并提取公钥
func ParseRSAPublicKeyFromCert(certPEM []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("decode PEM data failed")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.New("failed to parse certificate: " + err.Error())
	}

	rsaPub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("pem data is not a valid RSA public key")
	}

	return rsaPub, nil
}

// ParseRSAPrivateKey 从 ASN.1 PEM 格式的证书中解析 PKCS #1 格式或者 PKCS #8 格式的 RSA 私钥
func ParseRSAPrivateKey(certPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("decode PEM data failed")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return priv, nil
	}

	privPKCS8, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPriv, ok := privPKCS8.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("pem data is not a valid RSA public key")
	}

	return rsaPriv, nil
}

// ParseRSAPKCS1PrivateKey 从 ASN.1 PEM 格式的证书中解析 PKCS #1 格式的 RSA 私钥，PEM 块类型通常是 "RSA PRIVATE KEY"
func ParseRSAPKCS1PrivateKey(certPem []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(certPem)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("decode PEM data failed")
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// ParseRSAPKCS8PrivateKey 从 ASN.1 PEM 格式的证书中解析 PKCS #8 格式的 RSA 私钥，PEM 块类型通常是 "PRIVATE KEY"
func ParseRSAPKCS8PrivateKey(certPem []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(certPem)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, errors.New("decode PEM data failed")
	}

	priv, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPriv, ok := priv.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("pem data is not a valid RSA private key")
	}

	return rsaPriv, nil
}

// ParsePKCS1PrivateKey 从 ASN.1 PEM 格式的证书中解析 PKCS #8 格式的私钥，PEM 块类型通常是 "PRIVATE KEY"
func ParsePKCS8PrivateKey(certPem []byte) (any, error) {
	block, _ := pem.Decode(certPem)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, errors.New("decode PEM data failed")
	}

	return x509.ParsePKCS8PrivateKey(block.Bytes)
}

// RSAKeyGenerator 生成 RSA 密钥对
// 密钥长度推荐使用 2048 位或者更高
// 私钥格式和公钥格式可以随意组合
type RSAKeyGenerator struct {
	privKey *rsa.PrivateKey
	genErr  error
}

// NewRSAKeyGenerator 生成 RSA 密钥对
func NewRSAKeyGenerator(bits int) *RSAKeyGenerator {
	privKey, genErr := rsa.GenerateKey(rand.Reader, bits)
	return &RSAKeyGenerator{privKey, genErr}
}

// NewRSAKeyGenerator1024 生成 1024 位的 RSA 密钥对，密钥长度推荐使用 2048 位或者更高
func NewRSAKeyGenerator1024() *RSAKeyGenerator {
	privKey, genErr := rsa.GenerateKey(rand.Reader, 1024)
	return &RSAKeyGenerator{privKey, genErr}
}

// NewRSAKeyGenerator2048 生成 2048 位的 RSA 密钥对
func NewRSAKeyGenerator2048() *RSAKeyGenerator {
	privKey, genErr := rsa.GenerateKey(rand.Reader, 2048)
	return &RSAKeyGenerator{privKey, genErr}
}

// NewRSAKeyGenerator3072 生成 3072 位的 RSA 密钥对
func NewRSAKeyGenerator4096() *RSAKeyGenerator {
	privKey, genErr := rsa.GenerateKey(rand.Reader, 4096)
	return &RSAKeyGenerator{privKey, genErr}
}

// PKCS1PublicKeyDER 生成 PKCS #1 标准 DER 格式的 RSA 公钥
func (r *RSAKeyGenerator) PKCS1PublicKeyDER() ([]byte, error) {
	if r.genErr != nil {
		return nil, r.genErr
	}

	return x509.MarshalPKCS1PublicKey(&r.privKey.PublicKey), nil
}

// PKCS1PublicKeyPEM 生成 PKCS #1 标准 PEM 格式的 RSA 公钥
func (r *RSAKeyGenerator) PKCS1PublicKeyPEM() ([]byte, error) {
	pubDER, err := r.PKCS1PublicKeyDER()
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubDER,
	}), nil
}

// PKIXPublicKeyDER 生成 PKIX 标准 DER 格式的 RSA 公钥
func (r *RSAKeyGenerator) PKIXPublicKeyDER() ([]byte, error) {
	if r.genErr != nil {
		return nil, r.genErr
	}

	return x509.MarshalPKIXPublicKey(&r.privKey.PublicKey)
}

// PKIXPublicKeyPEM 生成 PKIX 标准 PEM 格式的 RSA 公钥
func (r *RSAKeyGenerator) PKIXPublicKeyPEM() ([]byte, error) {
	pubDER, err := r.PKIXPublicKeyDER()
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubDER,
	}), nil
}

// PKCS1PrivateKeyDER 生成 PKCS #1 标准 DER 格式的 RSA 私钥
func (r *RSAKeyGenerator) PKCS1PrivateKeyDER() ([]byte, error) {
	if r.genErr != nil {
		return nil, r.genErr
	}

	return x509.MarshalPKCS1PrivateKey(r.privKey), nil
}

// PKCS1PrivateKey 生成 PKCS #1 标准 PEM 格式的 RSA 私钥
func (r *RSAKeyGenerator) PKCS1PrivateKeyPEM() ([]byte, error) {
	privDER, err := r.PKCS1PrivateKeyDER()
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privDER,
	}), nil
}

// PKCS8PrivateKeyDER 生成 PKCS #8 标准 DER 格式的 RSA 私钥
func (r *RSAKeyGenerator) PKCS8PrivateKeyDER() ([]byte, error) {
	if r.genErr != nil {
		return nil, r.genErr
	}

	return x509.MarshalPKCS8PrivateKey(r.privKey)
}

// PKCS8PrivateKeyPEM 生成 PKCS #8 标准 PEM 格式的 RSA 私钥
func (r *RSAKeyGenerator) PKCS8PrivateKeyPEM() ([]byte, error) {
	privDER, err := r.PKCS8PrivateKeyDER()
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privDER,
	}), nil
}
