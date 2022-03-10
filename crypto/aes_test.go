package crypto

import (
	"crypto/aes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"testing"
)

func TestCBCEncrypt(t *testing.T) {
	key := []byte("df8dfs4dfs7e1231")
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}
	iv = []byte("1234567891234567")

	aesCipher := NewAES(key, WithIV(iv), WithPaddingMode(PKCS7Padding))
	cipherTxt, err := aesCipher.Encrypt([]byte("fsdf"))
	fmt.Println(base64.StdEncoding.EncodeToString(cipherTxt), err)
}

func TestCBCDecrypt(t *testing.T) {
	key := []byte("df8dfs4dfs7e1231")
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}
	iv = []byte("1234567891234567")

	aesCipher := NewAES(key, WithIV(iv), WithPaddingMode(PKCS7Padding))

	cipherTxt := "1+s5RyBQ91oAb5Q8dAB7rA=="
	data, _ := base64.StdEncoding.DecodeString(cipherTxt)
	plainTxt, err := aesCipher.Decrypt(data)
	fmt.Println(string(plainTxt), err)
}

//func TestTripleDES(t *testing.T) {
//	ci := NewCipher(TripleDESCBCPKCS5Padding)
//	cipherTxt := "oojNNJGXQdpryTj31ASbGNo9EwH4WDAJEo9RZRl7Huc51ZSDX8jJ+zlkxYIHLAlnxSOLod1qXVr6mqc0ObX+4INIO2cp5XyMsBmKlA2kGMg1OqPFQtCxrc4C7+VB5u9Cl9Jk+tvQqpT6P6lDGyarJ6q/lpl8RZEDfrMRVDEYkY7E2DZn+eRZ99m6QE6WcL047/Tb5EqTx2VHXhzlx4DVWa6akMej2hi8bORHSjzRhL4yf22o+VsRBMjUZR92QA61Z5Gs7NUnEThzAPKClvC8jYSZtgy9GL+tQOjQtH4ULu2iR4y6l3IwtR1JG9OK20wPdu2gT3AuwnHVcnr7S/jp9eZL1La8hlkPusN8cXwVo/+fUvc5jkJffN7YnXkUT5VgtfTtD+d7h8Pn98rCB+9RnwhV/n6novfkcZTZx7dvv/kjTBerR1sFxgo+dKrFneMmq7YUb5l1OV7dYGEejgKXOdnNS8up3mFtd23iHsA0j3yj7MbjS6iwHTamTdX+QVdzj6HfRYcXyiID8tL6Tx+Pd3jS+VrKm4BvPiYZ80j0AloT4druT8sRjdo2WU+vU0/MCKkb++zShpXr2Nkm5RwpQgar1g88qbo1z2BS+Cbug706wQydqpmlbiIGRGKoRrZcm5efvvCXRIChqQkEVztQMHt8wmDJqB3a1icXmJALWkqK6oAUf+3vL9S3C77SHnYM0jK4FnCrpPA4uXzAu9wbpb9044hfFwUF42SazIAx2VCF4nzzQjZPwPe0Ibc3KDwl7iPRrvzUOUGvnfmpj3BLIA9TcPMvjJTUYps3uHarcthoJFXjCOHzxVE0RnBOF3npmaKVLJfraeZyzHtpZR7ex2CRSJtP4Pq/NJoU48E2fds5vZQ/lDkL0w=="
//	fmt.Println(len(cipherTxt))
//	data, err := base64.StdEncoding.DecodeString(cipherTxt)
//	if err != nil {
//		println(err)
//		return
//	}
//	fmt.Println(len(data))
//
//	data, err = ci.DecryptWithIV([]byte("spring20140530airlines!)"), []byte("12345678"), data)
//	fmt.Println(string(data), err)
//}
