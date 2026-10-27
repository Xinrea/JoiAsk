package imagefile

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestRasterExtension(t *testing.T) {
	for _, test := range []struct {
		name, content, extension string
	}{
		{"png", "\x89PNG\r\n\x1a\n", ".png"},
		{"jpeg", "\xff\xd8\xff", ".jpg"},
		{"gif87", "GIF87a", ".gif"},
		{"gif89", "GIF89a", ".gif"},
		{"webp", "RIFF\x00\x00\x00\x00WEBPVP8 ", ".webp"},
		{"html", "<script>alert(1)</script>", ""},
		{"svg", `<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"/>`, ""},
		{"xml svg", `<?xml version="1.0"?><svg/>`, ""},
		{"javascript", "alert(1)", ""},
		{"empty", "", ""},
		{"unknown", "arbitrary bytes", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := RasterExtension([]byte(test.content)); got != test.extension {
				t.Fatalf("extension = %q, want %q", got, test.extension)
			}
		})
	}
}

func TestReadLimits(t *testing.T) {
	for _, size := range []int{0, 1, MaxSize, MaxSize + 1, MaxSize + 100} {
		reader := bytes.NewReader(make([]byte, size))
		content, err := Read(reader)
		valid := size > 0 && size <= MaxSize
		if (err == nil) != valid {
			t.Fatalf("size %d: error = %v", size, err)
		}
		if valid && len(content) != size {
			t.Fatalf("size %d: read %d bytes", size, len(content))
		}
		if int64(size)-int64(reader.Len()) > MaxSize+1 {
			t.Fatal("read beyond the size limit")
		}
	}
	if _, err := Read(errorReader{}); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("read error = %v", err)
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
