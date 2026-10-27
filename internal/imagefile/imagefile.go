// Package imagefile defines the shared size and content-type policy for stored images.
package imagefile

import (
	"errors"
	"io"
	"net/http"
)

const MaxSize = 5 << 20

// Read bounds the actual bytes read, rather than trusting a declared file size.
func Read(reader io.Reader) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(reader, MaxSize+1))
	if err != nil {
		return nil, err
	}
	if len(content) == 0 || len(content) > MaxSize {
		return nil, errors.New("图片不能为空且不能超过 5 MiB")
	}
	return content, nil
}

// RasterExtension derives a canonical, non-executable extension from the bytes.
// Never trust a client filename or Content-Type. SVG is intentionally excluded:
// unlike raster images it can execute scripts when opened as a document.
func RasterExtension(content []byte) string {
	switch http.DetectContentType(content) {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}
