// package syncx 提供了并发编程相关的标准库扩展功能
package syncx

import "runtime"

// RecoverHandler recover 函数的简单包装
func RecoverHandler(f func(panicErr interface{}, stack []byte)) {
	if err := recover(); err != nil {
		buf := make([]byte, 64<<10)
		n := runtime.Stack(buf, false)
		buf = buf[:n]

		f(err, buf)
	}
}

// SafeGo 使用 recover 恢复 goroutine 崩溃，并提供了一个崩溃处理函数
func SafeGo(fn func(), handler func(panicErr interface{}, stack []byte)) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				buf := make([]byte, 64<<10)
				n := runtime.Stack(buf, false)
				buf = buf[:n]

				handler(err, buf)
			}
		}()

		fn()
	}()
}
