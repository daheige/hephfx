package gutils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Uuid 生成 version4 的uuid
// 返回格式:eba1e8cd0460491049c644bdf3cf024d
func Uuid() string {
	u, err := uuid.NewRandom()
	if err != nil {
		return strings.Replace(RndUUID(), "-", "", -1)
	}

	return strings.Replace(u.String(), "-", "", -1)
}

// RndUUID realizes unique uuid based on time ns and random number
// There is no duplication of uuid on a single machine
// If you want to generate non-duplicate uuid on a distributed architecture
// Just add some custom strings in front of rndStr
// Return format: eba1e8cd-0460-4910-49c6-44bdf3cf024d
func RndUUID() string {
	s := RndUUIDMd5()
	return fmt.Sprintf("%s-%s-%s-%s-%s", s[:8], s[8:12], s[12:16], s[16:20], s[20:])
}

// RndUUIDMd5 make an md5 uuid
func RndUUIDMd5() string {
	ns := time.Now().UnixNano()
	rndStr := strings.Join([]string{
		strconv.FormatInt(ns, 10), strconv.FormatInt(RandInt64(1000, 9999), 10),
	}, "")

	return Md5(rndStr)
}

// RandInt64 crete a num [m,n]
func RandInt64(min, max int64) int64 {
	if min >= max || min == 0 || max == 0 {
		return max
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return r.Int63n(max-min) + min
}

// Md5 md5 func
func Md5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

// CatchStack 捕获指定stack信息,一般在处理panic/recover中处理
// 返回完整的堆栈信息和函数调用信息
func CatchStack() []byte {
	return debug.Stack()
}
