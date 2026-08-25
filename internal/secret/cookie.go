package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"

	"github.com/spf13/viper"
)

var ErrEncryptionKeyMissing = errors.New("未配置 B 站 Cookie 加密密钥")

func cookieAEAD() (cipher.AEAD, error) {
	configured := viper.GetString("security.bilibili_cookie_key")
	if len(configured) < 32 {
		return nil, ErrEncryptionKeyMissing
	}
	key := sha256.Sum256([]byte(configured))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func EncryptCookie(value string) (string, error) {
	aead, err := cookieAEAD()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, []byte(value), nil)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func DecryptCookie(value string) (string, error) {
	aead, err := cookieAEAD()
	if err != nil {
		return "", err
	}
	raw, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil || len(raw) < aead.NonceSize() {
		return "", errors.New("加密 Cookie 无效")
	}
	plain, err := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], nil)
	if err != nil {
		return "", errors.New("无法解密 B 站 Cookie")
	}
	return string(plain), nil
}
