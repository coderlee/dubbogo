/******************************************************
# DESC    : dubbogo constants
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-06-22 21:30
# FILE    : misc.go
******************************************************/

package common

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	VERSION string = "0.0.1"
	NAME    string = "dubbogo"
	DUBBO   string = "dubbo"
)

const (
	DUBBOGO_CTX_KEY string = "dubbogo-ctx"
)

const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var (
	src = rand.NewSource(time.Now().UnixNano())
)

func TimeSecondDuration(sec int) time.Duration {
	return time.Duration(sec) * time.Second
}

func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}

	return false
}

/*
 code example:

 data := []string{"one", "two", "three"}
 ArrayRemoveAt(&data, 2)
 fmt.Println("data len:", len(data), ", data:", data)

 data2 := []int32{1, 2, 3}
 ArrayRemoveAt(&data2, 2)
 fmt.Println("data2 len:", len(data2), ", data2:", data2)
*/
func ArrayRemoveAt(a interface{}, i int) {
	if i < 0 {
		return
	}

	if array, ok := a.(*[]int); ok {
		if len(*array) <= i {
			return
		}
		s := *array
		// s = append(s[:i], s[i+1:]...) // perfectly fine if i is the last element
		// *array = s
		*array = append(s[:i], s[i+1:]...) // perfectly fine if i is the last element
	} else if array, ok := a.(*[]string); ok {
		if len(*array) <= i {
			return
		}
		s := *array
		// s = append(s[:i], s[i+1:]...)
		// *array = s
		*array = append(s[:i], s[i+1:]...)
	}
}

func Future(sec int, f func()) {
	time.AfterFunc(TimeSecondDuration(sec), f)
}

func TrimPrefix(s string, prefix string) string {
	if strings.HasPrefix(s, prefix) {
		s = s[len(prefix):]
	}
	return s
}

func TrimSuffix(s string, suffix string) string {
	if strings.HasSuffix(s, suffix) {
		s = s[:len(s)-len(suffix)]
	}
	return s
}

func Goid() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}
	return id
}

func RandStringBytesMaskImprSrc(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
