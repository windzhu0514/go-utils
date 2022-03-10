package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"math/big"

	"github.com/windzhu0514/go-utils/crypto/rsa_ext"
)

// RSAEncryptPKCS1v15 RSA公钥加密 支持长文本加密
func RSAEncryptPKCS1v15(plainText, publicKey []byte) ([]byte, error) {
	rsaPub, err := ParseRSAPublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	var buff bytes.Buffer

	hash := sha256.New()
	blockSize := rsaPub.Size() - 2*hash.Size() - 2
	for s := 0; s < len(plainText); s += blockSize {
		e := s + blockSize
		if e > len(plainText) {
			e = len(plainText)
		}

		cipherText, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPub, plainText[s:e])
		if err != nil {
			return nil, err
		}

		buff.Write(cipherText)
	}

	return buff.Bytes(), nil
}

// RSADecryptPKCS1v15 RSA私钥解密 支持长文本加密
func RSADecryptPKCS1v15(cipherText, privateKey []byte) ([]byte, error) {
	rsaPriv, err := ParsePrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	var buff bytes.Buffer

	blockSize := rsaPriv.PublicKey.Size()
	for s := 0; s < len(cipherText); s += blockSize {
		e := s + blockSize
		if e > len(cipherText) {
			e = len(cipherText)
		}

		plainText, err := rsa.DecryptPKCS1v15(rand.Reader, rsaPriv, cipherText[s:e])
		if err != nil {
			return nil, err
		}

		buff.Write(plainText)
	}

	return buff.Bytes(), nil
}

// RSAEncryptOAEP RSA RSAES-OAEP加密方案是RFC3447推荐新应用使用的标准，
// RSAES-PKCS1-v1_5 是为了与已存在的应用兼容，并且不建议用于新应用。
// 详细参考：https://www.rfc-editor.org/rfc/rfc3447.html#section-7
func RSAEncryptOAEP(plainText, publicKey []byte) ([]byte, error) {
	rsaPub, err := ParseRSAPublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	var buff bytes.Buffer

	hash := sha256.New()
	blockSize := rsaPub.Size() - 2*hash.Size() - 2
	for s := 0; s < len(plainText); s += blockSize {
		e := s + blockSize
		if e > len(plainText) {
			e = len(plainText)
		}

		cipherText, err := rsa.EncryptOAEP(hash, rand.Reader, rsaPub, plainText[s:e], nil)
		if err != nil {
			return nil, err
		}

		buff.Write(cipherText)
	}

	return buff.Bytes(), nil
}

// RSADecryptOAEP RSA私钥解密
func RSADecryptOAEP(cipherText, privateKey []byte) ([]byte, error) {
	rsaPriv, err := ParsePrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	var buff bytes.Buffer

	blockSize := rsaPriv.PublicKey.Size()
	for s := 0; s < len(cipherText); s += blockSize {
		e := s + blockSize
		if e > len(cipherText) {
			e = len(cipherText)
		}

		plainText, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, rsaPriv, cipherText[s:e], nil)
		if err != nil {
			return nil, err
		}

		buff.Write(plainText)
	}

	return buff.Bytes(), nil
}

// RSAPrivateEncrypt RSA私钥加密
func RSAPrivateEncrypt(plainText, privateKey []byte) ([]byte, error) {
	priv, err := ParsePrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	return rsa_ext.PrivateKeyEncrypt(rand.Reader, priv, plainText)
}

// RSAPublicDecrypt RSA公钥解密
func RSAPublicDecrypt(cipherText, publicKey []byte) ([]byte, error) {
	pub, err := ParseRSAPublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	return rsa_ext.PublicKeyDecrypt(pub, cipherText)
}

func ParseRSAPublicKey(publickey []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(publickey)
	if block == nil || block.Type != "RSA PUBLIC KEY" {
		return nil, errors.New("failed to parse public key")
	}

	return x509.ParsePKCS1PublicKey(block.Bytes)
}

func ParseRSAPublicKeyFromCert(certPEM []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "RSA PUBLIC KEY" {
		return nil, errors.New("failed to parse public key")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.New("failed to parse certificate: " + err.Error())
	}

	pub := cert.PublicKey.(*rsa.PublicKey)
	return pub, nil
}

func ParsePrivateKey(privatekey []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(privatekey)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("failed to parse private key")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return priv, nil
	}

	keyPKCS8, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return keyPKCS8.(*rsa.PrivateKey), nil
}

// RSAEncryptNoPadding RSA无填充公钥加密
func RSAEncryptNoPadding(plaintext, publicKey []byte) ([]byte, error) {
	block, _ := pem.Decode(publicKey)
	if block == nil {
		return nil, errors.New("public key error")
	}

	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	pub := pubInterface.(*rsa.PublicKey)

	cipherText := plaintext
	m := new(big.Int).SetBytes(cipherText)
	e := big.NewInt(int64(pub.E))
	return new(big.Int).Exp(m, e, pub.N).Bytes(), nil
}

// TODO:RSADecryptNoPadding RSA无填充私钥解密
func RSADecryptNoPadding(ciphertext, privateKey []byte) ([]byte, error) {
	// block, _ := pem.Decode(privateKey)
	// if block == nil {
	// 	return nil, errors.New("private key error!")
	// }
	//
	// priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	// if err != nil {
	// 	return nil, err
	// }
	//
	return nil, nil
}
