package storage

import (
	"bytes"
	"sync"

	"github.com/spf13/viper"
)

var storage Storage
var once sync.Once

type Storage interface {
	Upload(filename string, content *bytes.Reader) (string, error)
	Delete(filename string) error
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

// Get the single storage instance
func Get() Storage {
	once.Do(func() {
		storeConfig := StorageConfig{
			StorageType: StrToType(viper.GetString("storage_type")),
			Address:     viper.GetString("oss.address"),
			Endpoint:    viper.GetString("oss.endpoint"),
			AccessKey:   viper.GetString("oss.access_key"),
			SecretKey:   viper.GetString("oss.secret_key"),
			Bucket:      viper.GetString("oss.bucket"),
		}
		storage = New(storeConfig)
	})
	return storage
}
