package storage

import (
	"bytes"
	"io/ioutil"
	"os"
)

type Local struct {
}

func (l *Local) Upload(filename string, content *bytes.Reader) (string, error) {
	data, err := ioutil.ReadAll(content)
	if err != nil {
		return "", err
	}
	err = os.WriteFile("frontend/public/upload-img/"+filename, data, 0644)
	if err != nil {
		return "", err
	}
	return "upload-img/" + filename, nil
}

func NewLocal() *Local {
	return &Local{}
}
