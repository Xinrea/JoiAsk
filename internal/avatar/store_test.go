package avatar

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

type fakeObjectStorage struct {
	uploadedFilename string
	uploadedContent  []byte
	deletedFilename  string
}

func (storage *fakeObjectStorage) Upload(filename string, content *bytes.Reader) (string, error) {
	storage.uploadedFilename = filename
	storage.uploadedContent, _ = io.ReadAll(content)
	return "upload-img/" + filename, nil
}

func (storage *fakeObjectStorage) Delete(filename string) error {
	storage.deletedFilename = filename
	return nil
}

func TestSaveAndDeleteBilibiliAvatar(t *testing.T) {
	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	objects := new(fakeObjectStorage)
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(png)), Request: request}, nil
	})}
	avatarStore := &Store{Client: client, Storage: objects}
	storedURL, err := avatarStore.Save(context.Background(), 123, "https://i0.hdslb.com/bfs/face/test.png")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(storedURL, "/upload-img/bilibili-avatar-123-") || len(objects.uploadedContent) != len(png) {
		t.Fatalf("avatar was not stored correctly: url=%q filename=%q", storedURL, objects.uploadedFilename)
	}
	if err := avatarStore.Delete(storedURL); err != nil {
		t.Fatal(err)
	}
	if objects.deletedFilename != objects.uploadedFilename {
		t.Fatalf("unexpected deleted object: %q", objects.deletedFilename)
	}
}

func TestRejectsNonBilibiliAvatarURL(t *testing.T) {
	_, err := (&Store{Client: http.DefaultClient, Storage: new(fakeObjectStorage)}).Save(context.Background(), 123, "https://example.com/avatar.png")
	if err == nil {
		t.Fatal("non-Bilibili avatar URL should be rejected")
	}
}
