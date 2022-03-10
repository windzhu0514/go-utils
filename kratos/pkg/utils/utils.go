package utils

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"strings"
	"time"
)

func MD5(src string) string {
	h := md5.New()
	_, _ = h.Write([]byte(src))
	return hex.EncodeToString(h.Sum([]byte("")))
}

func JsonMarshalString(v interface{}) string {
	return string(JsonMarshalByte(v))
}

func JsonMarshalByte(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		//logger.Error("utils.JsonMarshalByte:" + err.Error())
		return nil
	}

	return data
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

func TransactionID() string {
	return fmt.Sprintf("%02x", time.Now().Unix()) + fmt.Sprintf("%02x", rand.Intn(1000000))
}

type ComparableDate []string

func (d ComparableDate) Len() int {
	return len(d)
}

func (d ComparableDate) Less(i, j int) bool {
	di, _ := time.Parse("2006-01-02", d[i])
	dj, _ := time.Parse("2006-01-02", d[j])
	return di.Before(dj)
}

func (d ComparableDate) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func GetOutBoundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return ""
	}
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return strings.Split(localAddr.String(), ":")[0]
}

func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func JoinURLPath(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

type GroupError struct {
	errs []error
}

func (g *GroupError) Add(err error) {
	if err != nil {
		g.errs = append(g.errs, err)
	}
}

func (g *GroupError) AsError() error {
	var errMsg string
	for i, err := range g.errs {
		if i > 0 {
			errMsg += "|"
		}
		errMsg += fmt.Sprintf("error %d: %s", i+1, err.Error())
	}

	if errMsg == "" {
		return nil
	}

	return errors.New(errMsg)
}

// CompareHideString 比较strref 和strcheck  strcheck为星号字符串 支持中文
// 350204199806138023 -> 3502************23 令狐冲 -> 令*冲
func CompareHideString(full, hide string) bool {
	if full == "" && hide == "" {
		return true
	}

	if full == "" || hide == "" {
		return false
	}

	s1 := []rune(full)
	s2 := []rune(hide)

	if len(s1) != len(s2) {
		return false
	}

	for i := 0; i < len(s1); i++ {
		if string(s2[i]) == "*" {
			continue
		}

		if s1[i] == s2[i] {
			continue
		}

		return false
	}

	return true
}
