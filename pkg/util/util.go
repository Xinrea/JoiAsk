package util

import (
	"crypto/md5"
	"encoding/hex"
)

func Md5v(v string) string {
	d := []byte(v)
	m := md5.New()
	m.Write(d)
	return hex.EncodeToString(m.Sum(nil))
}
