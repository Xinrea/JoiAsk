package storage

import (
	"bytes"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/sirupsen/logrus"
)

type OSS struct {
	address string
	client  *oss.Client
	bucket  *oss.Bucket
}

func NewOSS(address string, endpoint string, accessKey string, secretKey string, bucket string) *OSS {
	var err error
	ossClient, err := oss.New(endpoint, accessKey, secretKey)
	if err != nil {
		logrus.Fatal(err)
	}
	// 获取存储空间。
	ossBucket, err := ossClient.Bucket(bucket)
	if err != nil {
		logrus.Fatal(err)
	}
	return &OSS{
		address, ossClient, ossBucket,
	}
}

func (s *OSS) Upload(filename string, content *bytes.Reader) (string, error) {
	err := s.bucket.PutObject("upload-img/"+filename, content)
	if err != nil {
		logrus.Error(err)
		return "", err
	}
	return s.address + "/upload-img/" + filename, nil
}
