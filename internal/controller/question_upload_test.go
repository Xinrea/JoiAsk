package controller

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"joiask-backend/internal/database"
	"joiask-backend/internal/imagefile"
	"joiask-backend/pkg/util"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type questionTestFile struct {
	name, contentType string
	content           []byte
}

func questionUploadRouter(t *testing.T) *gin.Engine {
	t.Helper()
	setupRecordIDTestDB(t)
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("frontend/public/upload-img", 0755); err != nil {
		t.Fatal(err)
	}
	previous := viper.Get("storage_type")
	viper.Set("storage_type", "local")
	t.Cleanup(func() { viper.Set("storage_type", previous) })
	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-session-secret-that-is-long-enough"))))
	router.POST("/api/question", (&QuestionController{}).Post)
	return router
}

func questionMultipartRequest(t *testing.T, files []questionTestFile) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range map[string]string{"tag_id": "1", "content": "anonymous upload"} {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range files {
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", `form-data; name="files[]"; filename="`+file.name+`"`)
		header.Set("Content-Type", file.contentType)
		part, err := writer.CreatePart(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(file.content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/question", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func questionUploadResponse(t *testing.T, router http.Handler, request *http.Request) testAPIResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	var response testAPIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func TestQuestionUploadRejectsUnsafeFilesWithoutWrites(t *testing.T) {
	pngFile := questionTestFile{"good.png", "image/png", testPNG(t)}
	for _, test := range []struct {
		name  string
		files []questionTestFile
	}{
		{"html", []questionTestFile{{"audit.html", "text/html", []byte("<script>alert(1)</script>")}}},
		{"spoofed MIME and extension", []questionTestFile{{"audit.png", "image/png", []byte("<script>alert(1)</script>")}}},
		{"svg", []questionTestFile{{"audit.svg", "image/svg+xml", []byte(`<svg onload="alert(1)"/>`)}}},
		{"empty", []questionTestFile{{"empty.png", "image/png", nil}}},
		{"oversized", []questionTestFile{{"large.png", "image/png", append(testPNG(t), make([]byte, imagefile.MaxSize)...)}}},
		{"mixed batch", []questionTestFile{pngFile, {"audit.html", "text/html", []byte("<script>alert(1)</script>")}}},
		{"too many", []questionTestFile{pngFile, pngFile, pngFile, pngFile, pngFile, pngFile, pngFile}},
	} {
		t.Run(test.name, func(t *testing.T) {
			router := questionUploadRouter(t)
			response := questionUploadResponse(t, router, questionMultipartRequest(t, test.files))
			if response.Code != 400 {
				t.Fatalf("response = %+v", response)
			}
			entries, err := os.ReadDir("frontend/public/upload-img")
			if err != nil || len(entries) != 0 {
				t.Fatalf("unexpected storage writes: %v, %v", entries, err)
			}
			var count int64
			if err := database.DB.Model(&database.Question{}).Count(&count).Error; err != nil || count != 1 {
				t.Fatalf("unexpected database writes: count=%d, err=%v", count, err)
			}
		})
	}
}

func TestQuestionUploadDerivesExtensionAndPreservesTextPosts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		router := questionUploadRouter(t)
		content := testPNG(t)
		for _, filename := range []string{"audit.html", "audit.SVG", "no-extension", "image.jpeg"} {
			response := questionUploadResponse(t, router, questionMultipartRequest(t, []questionTestFile{{filename, "text/html", content}}))
			if response.Code != 200 {
				t.Fatalf("%s: %+v", filename, response)
			}
			synctest.Wait()
			var question database.Question
			if err := database.DB.Last(&question).Error; err != nil {
				t.Fatal(err)
			}
			want := "upload-img/" + util.Md5v(string(content)) + ".png"
			if question.Images != want || question.ImagesNum != 1 {
				t.Fatalf("unexpected attachment: %+v", question)
			}
			stored, err := os.ReadFile("frontend/public/" + want)
			if err != nil || !bytes.Equal(content, stored) {
				t.Fatalf("stored image differs: %v", err)
			}
		}
		for _, request := range []*http.Request{
			questionMultipartRequest(t, nil),
			httptest.NewRequest(http.MethodPost, "/api/question", strings.NewReader("tag_id=1&content=text")),
		} {
			if request.Header.Get("Content-Type") == "" {
				request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			if response := questionUploadResponse(t, router, request); response.Code != 200 {
				t.Fatalf("text post: %+v", response)
			}
			synctest.Wait()
		}
	})
}

func TestQuestionUploadRejectsMalformedAndOversizedBodies(t *testing.T) {
	router := questionUploadRouter(t)
	malformed := httptest.NewRequest(http.MethodPost, "/api/question", strings.NewReader("--broken\r\n"))
	malformed.Header.Set("Content-Type", "multipart/form-data; boundary=broken")
	if response := questionUploadResponse(t, router, malformed); response.Code != 400 {
		t.Fatalf("malformed: %+v", response)
	}

	// Use an unknown Content-Length too: the limit must apply while reading.
	large := questionMultipartRequest(t, []questionTestFile{{"large.png", "image/png", make([]byte, 30<<20)}})
	large.ContentLength = -1
	if response := questionUploadResponse(t, router, large); response.Code != 413 {
		t.Fatalf("oversized: %+v", response)
	}
	entries, err := os.ReadDir("frontend/public/upload-img")
	if err != nil || len(entries) != 0 {
		t.Fatalf("unexpected storage writes: %v, %v", entries, err)
	}
	if large.MultipartForm != nil {
		t.Fatal("oversized request retained a multipart form")
	}
}

func TestSiteAssetKeepsExplicitSVGException(t *testing.T) {
	if got := assetExtension([]byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)); got != ".svg" {
		t.Fatalf("SVG extension = %q", got)
	}
	if got := assetExtension([]byte("<script>alert(1)</script>")); got != "" {
		t.Fatalf("HTML extension = %q", got)
	}
	if got := assetExtension(testPNG(t)); got != ".png" {
		t.Fatalf("PNG extension = %q", got)
	}
}
