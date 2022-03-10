package stdx

import "time"

// 解析中文时间
// github.com/WindomZ/timezh

func UnixFromMilliSec(milliSec int64) time.Time {
	return time.Unix(milliSec/1000, milliSec%1000*1e6)
}
