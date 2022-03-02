package storage

import (
	"bytes"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var client *oss.Client
var bucket *oss.Bucket

func Init() {
	var err error
	client, err = oss.New(viper.GetString("oss.endpoint"), viper.GetString("oss.access_key"), viper.GetString("oss.secret_key"))
	if err != nil {
		logrus.Fatal(err)
	}
	// 获取存储空间。
	bucket, err = client.Bucket(viper.GetString("oss.bucket"))
	if err != nil {
		logrus.Fatal(err)
	}
}

func Upload(filename string, content *bytes.Reader) error {
	err := bucket.PutObject(filename, content)
	if err != nil {
		logrus.Error(err)
		return err
	}
	return nil
}
