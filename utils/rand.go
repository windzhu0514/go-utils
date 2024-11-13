package utils

import (
	"fmt"
	"math/rand"
	"time"
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

// GenFakeMobile 生成假手机号
func GenFakeMobile() string {
	MobileNOPrefix := [...]string{"187", "156", "189", "186", "137", "139", "135", "157", "188", "153", "183", "131", "177"}
	rand.Seed(time.Now().UnixNano())
	mobile := MobileNOPrefix[rand.Int()%len(MobileNOPrefix)]
	mobile = mobile + fmt.Sprintf("%08d", rand.Int63n(99999999))

	return mobile
}

// GenFakeEmail 生成假的email地址
func GenFakeEmail(prefix string) string {
	if prefix == "" {
		prefix = GenFakeMobile()
	}

	mailDomains := []string{"163.com", "126.com", "sina.com.cn", "139.com", "yeah.net", "21cn.com", "sohu.com", "qq.com"}

	index := rand.Intn(len(mailDomains))

	return prefix + "@" + mailDomains[index]
}
