package util

import "time"

func GetMillis() int64 {
	return GetNanos() / 1000000
}

func GetNanos() int64 {
	return time.Now().UnixNano()
}
