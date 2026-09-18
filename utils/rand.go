package utils

import (
	"math/rand"
)

// 从source里随机字符生成出长度为n的字符串
func RandStringN(n int, source string) (str string) {
	len := len(source)
	if len == 0 {
		return
	}

	for i := 0; i < n; i++ {
		str += string(source[rand.Intn(len)])
	}

	return
}
