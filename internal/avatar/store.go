package avatar

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"joiask-backend/internal/database"
	"joiask-backend/internal/storage"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

const maxAvatarSize = 5 << 20

type Store struct {
	Client  *http.Client
	Storage storage.Storage
}

func NewStore() *Store {
	return &Store{Client: &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(request *http.Request, _ []*http.Request) error {
			if !isAllowedBilibiliImageURL(request.URL) {
				return errors.New("B 站头像重定向地址无效")
			}
			return nil
		},
	}}
}

func isAllowedBilibiliImageURL(value *url.URL) bool {
	host := strings.ToLower(value.Hostname())
	return value.Scheme == "https" && (host == "hdslb.com" || strings.HasSuffix(host, ".hdslb.com"))
}

func (s *Store) Save(ctx context.Context, uid int64, remoteURL string) (string, error) {
	parsed, err := url.Parse(remoteURL)
	if err != nil || !isAllowedBilibiliImageURL(parsed) {
		return "", errors.New("B 站头像地址无效")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; JoiAsk/1.0)")
	resp, err := s.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载 B 站头像失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !isAllowedBilibiliImageURL(resp.Request.URL) {
		return "", fmt.Errorf("下载 B 站头像失败: HTTP %d", resp.StatusCode)
	}
	content, err := io.ReadAll(io.LimitReader(resp.Body, maxAvatarSize+1))
	if err != nil {
		return "", fmt.Errorf("读取 B 站头像失败: %w", err)
	}
	if len(content) == 0 || len(content) > maxAvatarSize {
		return "", errors.New("B 站头像大小无效")
	}
	contentType := http.DetectContentType(content)
	extensions, _ := mime.ExtensionsByType(contentType)
	if !strings.HasPrefix(contentType, "image/") || len(extensions) == 0 {
		return "", errors.New("B 站头像格式无效")
	}
	extension := extensions[0]
	if contentType == "image/jpeg" {
		extension = ".jpg"
	}
	hash := sha256.Sum256(content)
	filename := "bilibili-avatar-" + strconv.FormatInt(uid, 10) + "-" + hex.EncodeToString(hash[:8]) + extension
	objectStorage := s.Storage
	if objectStorage == nil {
		objectStorage = storage.Get()
	}
	storedURL, err := objectStorage.Upload(filename, bytes.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("保存 B 站头像失败: %w", err)
	}
	if !strings.Contains(storedURL, "://") && !strings.HasPrefix(storedURL, "/") {
		storedURL = "/" + storedURL
	}
	return storedURL, nil
}

func (s *Store) Delete(storedURL string) error {
	if storedURL == "" {
		return nil
	}
	parsed, err := url.Parse(storedURL)
	if err != nil {
		return nil
	}
	marker := "/upload-img/"
	if !strings.Contains(parsed.Path, marker) && !strings.HasPrefix(parsed.Path, "upload-img/") {
		return nil
	}
	filename := path.Base(parsed.Path)
	if filename == "." || filename == "/" || filename == "" {
		return nil
	}
	objectStorage := s.Storage
	if objectStorage == nil {
		objectStorage = storage.Get()
	}
	return objectStorage.Delete(filename)
}

func (s *Store) SyncExistingUsers(ctx context.Context) {
	var users []database.User
	if err := database.DB.Where("bilibili_avatar <> '' AND bilibili_avatar NOT LIKE ?", "%/upload-img/%").Find(&users).Error; err != nil {
		return
	}
	for _, user := range users {
		select {
		case <-ctx.Done():
			return
		default:
		}
		storedURL, err := s.Save(ctx, user.BilibiliUID, user.BilibiliAvatar)
		if err != nil {
			continue
		}
		result := database.DB.Model(&database.User{}).
			Where("bilibili_uid = ? AND bilibili_avatar = ?", user.BilibiliUID, user.BilibiliAvatar).
			Update("bilibili_avatar", storedURL)
		if result.Error != nil || result.RowsAffected != 1 {
			_ = s.Delete(storedURL)
		}
	}
}
