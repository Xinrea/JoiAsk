package storage

import (
	"bytes"
)

type Storage interface {
	Upload(filename string, content *bytes.Reader) (string, error)
}

const (
	TYPE_LOCAL = iota
	TYPE_OSS
)

func StrToType(s string) int {
	switch s {
	case "local":
		return TYPE_LOCAL
	case "oss":
		return TYPE_OSS
	}
	panic("Invalid storage type")
}

type StorageConfig struct {
	StorageType int
	Address     string
	Endpoint    string
	AccessKey   string
	SecretKey   string
	Bucket      string
}

func New(s StorageConfig) Storage {
	switch s.StorageType {
	case TYPE_LOCAL:
		return NewLocal()
	case TYPE_OSS:
		return NewOSS(s.Address, s.Endpoint, s.AccessKey, s.SecretKey, s.Bucket)
	}
	return nil
}
