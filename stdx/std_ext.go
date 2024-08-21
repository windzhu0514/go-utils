package stdx

import (
	"reflect"
	"runtime"
)

// 解析中文时间
// github.com/WindomZ/timezh

// 模拟三元运算符（Ternary Operator"）
func Ter[T any](cond bool, a, b T) T {
	if cond {
		return a
	}

	return b
}

// 判断接口是否为 nil
func IsNil(x interface{}) bool {
	if x == nil {
		return true
	}

	return reflect.ValueOf(x).IsNil()
}

func SafeGo(fun func(), handler func(panicErr interface{}, stack []byte)) {
	defer func() {
		if err := recover(); err != nil {
			buf := make([]byte, 64<<10)
			n := runtime.Stack(buf, false)
			buf = buf[:n]

			handler(err, buf)
		}
	}()

	go fun()
}
