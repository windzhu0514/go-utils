package utils

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unsafe"
)

// EqualFloat64 比较float64 f1 f2可以是字符串或者float64
func EqualFloat64(f1 interface{}, f2 interface{}) (int, error) {
	ff1, err := Interface2Float64(f1)
	if err != nil {
		errMsg := fmt.Sprintf("parseFloat64 parse %v error:"+err.Error()+"\n", f1)
		return 0, errors.New(errMsg)
	}
	ff2, err := Interface2Float64(f2)
	if err != nil {
		errMsg := fmt.Sprintf("parseFloat64 parse %v error:"+err.Error()+"\n", f2)
		return 0, errors.New(errMsg)
	}

	if ff1-ff2 > 0.0 {
		return 1, nil
	} else if ff1-ff2 < 0.0 {
		return -1, nil
	} else {
		return 0, nil
	}
}

func Interface2Float64(v interface{}) (fv float64, err error) {
	switch vv := v.(type) {
	case string:
		fv, err = strconv.ParseFloat(vv, 64)
		if err != nil {
			return
		}
		return
	case float64:
		fv = vv
		return
	case float32:
		fv = float64(vv)
		return
	case int:
		fv = float64(vv)
		return
	case int64:
		fv = float64(vv)
		return
	default:
		return fv, errors.New("格式不正确")
	}
}

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

// 字符串和byte互转 无copy 无垃圾回收
func S2b(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

func B2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// 控制台等待动画
func Wait(timeout time.Duration) {
	timer := time.After(timeout)
	for {
		select {
		case <-timer:
			fmt.Printf("\r")
			return
		default:
		}

		for _, c := range "/-\\|" {
			fmt.Printf("\r%c", c)
			time.Sleep(time.Second)
		}
	}
}

// 是否是中文
func IsChinese(str string) bool {
	for _, v := range str {
		if !unicode.Is(unicode.Han, v) {
			return false
		}
	}

	return true
}

var delimiter = []byte("\n")

const base64MaxLenRFC2045 = 76

// Base64WrapRFC2045 返回符合 RFC 2045 的Base64 encoded结果(每76个字符添加\n)
func Base64WrapRFC2045(src []byte) (m string) {
	m = base64.StdEncoding.EncodeToString(src)
	the_len := len(m)

	if the_len <= base64MaxLenRFC2045 {
		return m
	}

	new_m := []byte(m)

	// set the slice capacity to the slice len + each newline delimiters
	m1 := make([]byte, 0, the_len+(len(delimiter)*int(the_len/base64MaxLenRFC2045)))
	ii := 0
	for i := 0; i < int(the_len/base64MaxLenRFC2045); i++ {
		m1 = append(m1, new_m[i*base64MaxLenRFC2045:(i+1)*base64MaxLenRFC2045]...)
		m1 = append(m1, delimiter...)
		ii++
	}
	m1 = append(m1, new_m[ii*base64MaxLenRFC2045:the_len]...)
	m = string(m1)
	return m
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

func LocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

// ExternalIP get external ip.
func ExternalIP() (res []string) {
	inters, err := net.Interfaces()
	if err != nil {
		return
	}
	for _, inter := range inters {
		if !strings.HasPrefix(inter.Name, "lo") {
			addrs, err := inter.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok {
					if ipnet.IP.IsLoopback() || ipnet.IP.IsLinkLocalMulticast() || ipnet.IP.IsLinkLocalUnicast() {
						continue
					}
					if ip4 := ipnet.IP.To4(); ip4 != nil {
						switch true {
						case ip4[0] == 10:
							continue
						case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
							continue
						case ip4[0] == 192 && ip4[1] == 168:
							continue
						default:
							res = append(res, ipnet.IP.String())
						}
					}
				}
			}
		}
	}
	return
}

// InternalIP get internal ip.
func InternalIP() string {
	inters, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, inter := range inters {
		if !strings.HasPrefix(inter.Name, "lo") {
			addrs, err := inter.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						return ipnet.IP.String()
					}
				}
			}
		}
	}
	return ""
}

func IsPowerOfTwo(x int) bool {
	return x&(x-1) == 0
}

func NameOfFunction(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

// IPv4 点分十进制与uint32的转换
func Uint32toIP4(ipInt uint32) string {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, ipInt)
	return ip.String()
}

func IP4toUint32(ipAddr string) uint32 {
	ip := net.ParseIP(ipAddr)
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func CompareHideString(first, second, hide string) bool {
	if first == "" && second == "" {
		return true
	}

	if first == "" || second == "" {
		return false
	}

	s1 := []rune(strings.ToUpper(first))
	s2 := []rune(strings.ToUpper(second))

	if len(s1) != len(s2) {
		return false
	}

	for i := 0; i < len(s1); i++ {
		if string(s1[i]) == hide || string(s2[i]) == hide {
			continue
		}

		if s1[i] == s2[i] {
			continue
		}

		return false
	}

	return true
}

func SQLXFields(values interface{}) []string {
	v := reflect.ValueOf(values)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var fields []string
	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			_ = f
			field := v.Type().Field(i).Tag.Get("db")
			if field != "" {
				fields = append(fields, field)
			}
		}
		return fields
	}

	if v.Kind() == reflect.Map {
		for _, keyv := range v.MapKeys() {
			fields = append(fields, keyv.String())
		}
		return fields
	}

	return nil
}
