package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Settings struct {
	Enabled            bool   `json:"enabled"`
	Host               string `json:"host"`
	Port               int    `json:"port"`
	Username           string `json:"username"`
	Password           string `json:"password"`
	PasswordConfigured bool   `json:"password_configured"`
	Security           string `json:"security"`
	FromEmail          string `json:"from_email"`
	FromName           string `json:"from_name"`
	SiteURL            string `json:"site_url"`
}

func NormalizeEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value || len(value) > 254 || strings.ContainsAny(value, "\r\n") || !strings.Contains(value, "@") {
		return "", errors.New("请输入有效的邮箱地址")
	}
	// Restrict to ASCII mailbox addresses; SMTPUTF8 is not supported by net/smtp.
	for _, r := range value {
		if r > 127 {
			return "", errors.New("邮箱地址需使用英文字符")
		}
	}
	return value, nil
}

func (s *Settings) Normalize() {
	s.Host = strings.TrimSpace(s.Host)
	s.Username = strings.TrimSpace(s.Username)
	s.FromEmail = strings.ToLower(strings.TrimSpace(s.FromEmail))
	s.FromName = strings.TrimSpace(s.FromName)
	s.SiteURL = strings.TrimRight(strings.TrimSpace(s.SiteURL), "/")
	if s.Port == 0 {
		s.Port = 587
	}
	if s.Security == "" {
		s.Security = "starttls"
	}
}

func (s Settings) Validate() error {
	if s.Host == "" || strings.ContainsAny(s.Host, " /\r\n\t") || s.Port < 1 || s.Port > 65535 {
		return errors.New("请填写有效的 SMTP 主机和端口")
	}
	if s.Security != "starttls" && s.Security != "tls" && s.Security != "none" {
		return errors.New("SMTP 加密方式无效")
	}
	if _, err := NormalizeEmail(s.FromEmail); err != nil {
		return errors.New("发件邮箱无效")
	}
	if strings.ContainsAny(s.FromName, "\r\n") {
		return errors.New("发件人名称无效")
	}
	if s.Username != "" && s.Security == "none" && s.Host != "localhost" && s.Host != "127.0.0.1" && s.Host != "::1" {
		return errors.New("远程 SMTP 认证需要使用 STARTTLS 或 TLS 加密")
	}
	if s.Username != "" && s.Password == "" {
		return errors.New("请填写 SMTP 密码或授权码")
	}
	u, err := url.Parse(s.SiteURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("网站地址需为完整的 HTTP 或 HTTPS 地址，且不能包含查询参数或片段")
	}
	return nil
}

func (s Settings) Link(purpose, token string) string {
	fragment := url.Values{"purpose": {purpose}, "token": {token}}
	return s.SiteURL + "/account/email/#" + fragment.Encode()
}

type Sender interface {
	Send(context.Context, Settings, string, string, string) error
}
type SMTPSender struct{}

func (SMTPSender) Send(ctx context.Context, settings Settings, to, subject, body string) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	if _, err := NormalizeEmail(to); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	address := net.JoinHostPort(settings.Host, strconv.Itoa(settings.Port))
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	rawConn := conn
	defer rawConn.Close()
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}
	stop := context.AfterFunc(ctx, func() { _ = rawConn.Close() })
	defer stop()
	tlsConfig := &tls.Config{ServerName: settings.Host, MinVersion: tls.VersionTLS12}
	if settings.Security == "tls" {
		secure := tls.Client(conn, tlsConfig)
		if err := secure.HandshakeContext(ctx); err != nil {
			return err
		}
		conn = secure
	}
	client, err := smtp.NewClient(conn, settings.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if settings.Security == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("SMTP server does not support STARTTLS")
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return err
		}
	}
	if settings.Username != "" {
		var auth smtp.Auth
		if ok, mechanisms := client.Extension("AUTH"); ok {
			// LOGIN is widely used by SMTP providers that do not advertise PLAIN.
			if strings.Contains(" "+mechanisms+" ", " PLAIN ") {
				auth = smtp.PlainAuth("", settings.Username, settings.Password, settings.Host)
			} else if strings.Contains(" "+mechanisms+" ", " LOGIN ") {
				auth = &loginAuth{settings.Username, settings.Password, settings.Host}
			}
		}
		if auth == nil {
			return errors.New("SMTP server does not support PLAIN or LOGIN authentication")
		}
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(settings.FromEmail); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	from := (&mail.Address{Name: settings.FromName, Address: settings.FromEmail}).String()
	_, err = fmt.Fprintf(writer, "From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n", from, to, mime.QEncoding.Encode("UTF-8", subject), time.Now().Format(time.RFC1123Z))
	if err != nil {
		return err
	}
	encoded := quotedprintable.NewWriter(writer)
	if _, err := io.WriteString(encoded, body); err != nil {
		return err
	}
	if err := encoded.Close(); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	// DATA acceptance determines delivery success; a failing QUIT must not revoke the link.
	_ = client.Quit()
	return nil
}

type loginAuth struct{ username, password, host string }

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !server.TLS && server.Name != "localhost" && server.Name != "127.0.0.1" && server.Name != "::1" {
		return "", nil, errors.New("SMTP authentication requires TLS")
	}
	if server.Name != a.host {
		return "", nil, errors.New("SMTP host mismatch")
	}
	return "LOGIN", nil, nil
}
func (a *loginAuth) Next(challenge []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	switch strings.ToLower(strings.TrimSpace(string(challenge))) {
	case "username:", "user name:":
		return []byte(a.username), nil
	case "password:":
		return []byte(a.password), nil
	default:
		return nil, errors.New("unexpected SMTP LOGIN challenge")
	}
}
