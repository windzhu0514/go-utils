package utils

import (
	"runtime"
	"time"
)

// 函数执行时间
// defer Elapsed()()
func Elapsed(f func(funcName string, elapsed time.Duration)) func() {
	start := time.Now()
	return func() {
		elapsed := time.Since(start)
		pc, _, _, ok := runtime.Caller(1)

		var funcName string
		if ok {
			f := runtime.FuncForPC(pc)
			funcName = f.Name()
		}

		// fmt.Println(funcName+" 耗时: ", elapsed)
		f(funcName, elapsed)
	}
}
