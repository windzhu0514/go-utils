package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"testing"
)

var pubPEMPKIX = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDR5d33b53ronRpKaY0anNBcrk3
HjgR0iDV9WxiU3lV1gIuTSlmQ4freqxNyqvaPzyq/OEV4pAvWvnZ7XWfhHOvCmqD
x7cOBILyLancCghQt1IZBLUcFNwQtb53QURHTolJsFjzZ6YuRuIEzUhjGxs4cOpK
996tJA02eia15DL6SwIDAQAB
-----END PUBLIC KEY-----`

var privPEMPKCS1 = `-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQDR5d33b53ronRpKaY0anNBcrk3HjgR0iDV9WxiU3lV1gIuTSlm
Q4freqxNyqvaPzyq/OEV4pAvWvnZ7XWfhHOvCmqDx7cOBILyLancCghQt1IZBLUc
FNwQtb53QURHTolJsFjzZ6YuRuIEzUhjGxs4cOpK996tJA02eia15DL6SwIDAQAB
AoGAI9knjNACX7EyQpe7bC5WGvJ2EaTWnKyPsRsmdLCfA3r2imPhUzbrattbvDmG
GlM8kFN6OdulFoFqNL8qnKDphMduMm12TxCZey7+XXAYFKcp0MfGB3YsNwuRAwKJ
KMvP34PtvniFMzlpIotx1vH4D73P9ZSB6FpR1Zd3yTBbayECQQD0kbMRR8gPcs9Y
VHSZ2gRr7QNAgPSU3WEaau3+zTYME0aqpMCXpwJavArotIIuZpDRkGnAgyqIRwr8
AlnrnOBRAkEA27VScn1Zmx67pkBry9CZeuNcCXCe+crsREyXyxhfQHqORKPhy7oy
ePuBQXMkAsyhczdP4WU71s0OZLNKbOWF2wJBAKJdnEkro5kF+rvEjgiaIfdYyaDo
O/gcpNu6A7j2mhwYLAEcEbRcrwZ38rIdLJQLibdnLZ4eNn101MkL8pCmfJECQEVK
mVcMW39FoQ7c0GJ2RoFwRS2g/DJxojQkJ9dgt2HBhS72tJapAZQQVgKLSrcrTFvm
/OVsJQQopbpypS1U8fMCQGnl+XW7ZlqME0maiGAVA6ueXhgWIKikkBtPPjKWZI7C
1Ab3ezfWzrrYYtx8ytFo9/vTc0tKjkzjHsbt7lvDas8=
-----END RSA PRIVATE KEY-----`

func TestRSAEncryptPKCS1v15(t *testing.T) {
	rsa, err := NewRSA(WithPublicKey([]byte(pubPEMPKIX)), WithPrivateKey([]byte(privPEMPKCS1)))
	if err != nil {
		t.Error(err)
	}

	cipherBytes, err := rsa.EncryptPKCS1v15Base64([]byte("hhliotwapntjcqlt"))
	if err != nil {
		t.Error(err)
	}

	fmt.Println(string(cipherBytes))

	plainBytes, err := rsa.DecryptPKCS1v15Base64(cipherBytes)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(string(plainBytes))
}

func TestRSAEncryptOAEP(t *testing.T) {
	rsa, err := NewRSA(WithPublicKey([]byte(pubPEMPKIX)), WithPrivateKey([]byte(privPEMPKCS1)))
	if err != nil {
		t.Error(err)
	}

	cipherBytes, err := rsa.EncryptOAEPBase64(sha256.New(), []byte("hhliotwapntjcqlt"), nil)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(string(cipherBytes))

	plainBytes, err := rsa.DecryptOAEPBase64(sha256.New(), cipherBytes, nil)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(string(plainBytes))
}

var certPEM = []byte(`-----BEGIN CERTIFICATE-----
MIIFazCCA1OgAwIBAgIJAL8a/lsnspOqMA0GCSqGSIb3DQEBCwUAMEwxCzAJBgNV
BAYTAlVLMRMwEQYDVQQIDApUZXN0LVN0YXRlMRUwEwYDVQQKDAxHb2xhbmcgVGVz
dHMxETAPBgNVBAMMCHRlc3QtZGlyMB4XDTE3MDIwMTIzNTAyN1oXDTI3MDEzMDIz
NTAyN1owTDELMAkGA1UEBhMCVUsxEzARBgNVBAgMClRlc3QtU3RhdGUxFTATBgNV
BAoMDEdvbGFuZyBUZXN0czERMA8GA1UEAwwIdGVzdC1kaXIwggIiMA0GCSqGSIb3
DQEBAQUAA4ICDwAwggIKAoICAQDzBoi43Yn30KN13PKFHu8LA4UmgCRToTukLItM
WK2Je45grs/axg9n3YJOXC6hmsyrkOnyBcx1xVNgSrOAll7fSjtChRIX72Xrloxu
XewtWVIrijqz6oylbvEmbRT3O8uynu5rF82Pmdiy8oiSfdywjKuPnE0hjV1ZSCql
MYcXqA+f0JFD8kMv4pbtxjGH8f2DkYQz+hHXLrJH4/MEYdVMQXoz/GDzLyOkrXBN
hpMaBBqg1p0P+tRdfLXuliNzA9vbZylzpF1YZ0gvsr0S5Y6LVtv7QIRygRuLY4kF
k+UYuFq8NrV8TykS7FVnO3tf4XcYZ7r2KV5FjYSrJtNNo85BV5c3xMD3fJ2XcOWk
+oD1ATdgAM3aKmSOxNtNItKKxBe1mkqDH41NbWx7xMad78gDznyeT0tjEOltN2bM
uXU1R/jgR/vq5Ec0AhXJyL/ziIcmuV2fSl/ZxT4ARD+16tgPiIx+welTf0v27/JY
adlfkkL5XsPRrbSguISrj7JeaO/gjG3KnDVHcZvYBpDfHqRhCgrosfe26TZcTXx2
cRxOfvBjMz1zJAg+esuUzSkerreyRhzD7RpeZTwi6sxvx82MhYMbA3w1LtgdABio
9JRqZy3xqsIbNv7N46WO/qXL1UMRKb1UyHeW8g8btboz+B4zv1U0Nj+9qxPBbQui
dgL9LQIDAQABo1AwTjAdBgNVHQ4EFgQUy0/0W8nwQfz2tO6AZ2jPkEiTzvUwHwYD
VR0jBBgwFoAUy0/0W8nwQfz2tO6AZ2jPkEiTzvUwDAYDVR0TBAUwAwEB/zANBgkq
hkiG9w0BAQsFAAOCAgEAvEVnUYsIOt87rggmLPqEueynkuQ+562M8EDHSQl82zbe
xDCxeg3DvPgKb+RvaUdt1362z/szK10SoeMgx6+EQLoV9LiVqXwNqeYfixrhrdw3
ppAhYYhymdkbUQCEMHypmXP1vPhAz4o8Bs+eES1M+zO6ErBiD7SqkmBElT+GixJC
6epC9ZQFs+dw3lPlbiZSsGE85sqc3VAs0/JgpL/pb1/Eg4s0FUhZD2C2uWdSyZGc
g0/v3aXJCp4j/9VoNhI1WXz3M45nysZIL5OQgXymLqJElQa1pZ3Wa4i/nidvT4AT
Xlxc/qijM8set/nOqp7hVd5J0uG6qdwLRILUddZ6OpXd7ZNi1EXg+Bpc7ehzGsDt
3UFGzYXDjxYnK2frQfjLS8stOQIqSrGthW6x0fdkVx0y8BByvd5J6+JmZl4UZfzA
m99VxXSt4B9x6BvnY7ktzcFDOjtuLc4B/7yg9fv1eQuStA4cHGGAttsCg1X/Kx8W
PvkkeH0UWDZ9vhH9K36703z89da6MWF+bz92B0+4HoOmlVaXRkvblsNaynJnL0LC
Ayry7QBxuh5cMnDdRwJB3AVJIiJ1GVpb7aGvBOnx+s2lwRv9HWtghb+cbwwktx1M
JHyBf3GZNSWTpKY7cD8V+NnBv3UuioOVVo+XAU4LF/bYUjdRpxWADJizNtZrtFo=
-----END CERTIFICATE-----
`)

func TestParseRSAPublicKeyFromCert(t *testing.T) {
	pub, err := ParseRSAPublicKeyFromCert(certPEM)

	fmt.Println(pub.Size(), err)
}

func RSAEncrypt2(plainText, certPEM []byte) ([]byte, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, errors.New("public key error")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.New("failed to parse certificate: " + err.Error())
	}

	pub := cert.PublicKey.(*rsa.PublicKey)

	return rsa.EncryptPKCS1v15(rand.Reader, pub, plainText)
}

func TestNewRSAKeyGenerator1024(t *testing.T) {
	g := NewRSAKeyGenerator1024()

	pkixKey, err := g.PKIXPublicKeyPEM()
	if err != nil {
		t.Error(err)
	}

	fmt.Println(string(pkixKey))

	pkcs1Key, err := g.PKCS1PrivateKeyPEM()
	if err != nil {
		t.Error(err)
	}

	fmt.Println(string(pkcs1Key))
}
