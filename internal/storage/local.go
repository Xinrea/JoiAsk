package storage

import (
	"bytes"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

type Local struct {
}

func (l *Local) Upload(filename string, content *bytes.Reader) (string, error) {
	data, err := io.ReadAll(content)
	if err != nil {
		return "", err
	}
	err = os.WriteFile("frontend/public/upload-img/"+filename, data, 0644)
	if err != nil {
		return "", err
	}
	return "upload-img/" + filename, nil
}

func (l *Local) Delete(filename string) error {
	err := os.Remove("frontend/public/upload-img/" + filename)
	if err != nil {
		logrus.Error("remove file failed: ", err)
	}
	return err
}

func NewLocal() *Local {
	return &Local{}
}
