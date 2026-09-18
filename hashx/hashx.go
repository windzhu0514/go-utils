package hashx

import (
	"crypto/md5"
	"encoding/hex"
	"hash"
)

type Hash16 interface {
	hash.Hash
	Sum16() uint16
}

func MD5HexString(src string) string {
	h := md5.New()
	_, _ = h.Write([]byte(src))
	return hex.EncodeToString(h.Sum([]byte("")))
}
