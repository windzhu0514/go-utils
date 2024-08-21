package utils

import (
	"crypto/md5"
	"encoding/hex"
)

func MD5Hex(src string) string {
	h := md5.New()
	_, _ = h.Write([]byte(src))
	return hex.EncodeToString(h.Sum([]byte("")))
}
