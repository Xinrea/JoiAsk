package mailer

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"testing"
)

func testSettings() Settings {
	return Settings{Enabled: true, Host: "localhost", Port: 587, Security: "starttls", FromEmail: "sender@example.test", FromName: "测试提问箱", SiteURL: "https://ask.example.test"}
}

func TestEmailAndSettingsValidation(t *testing.T) {
	for _, input := range []string{"abc", "Person <user@example.com>", "u@example.com\r\nBcc: other@example.com", "用户@example.com"} {
		if _, err := NormalizeEmail(input); err == nil {
			t.Fatalf("accepted invalid address %q", input)
		}
	}
	normalized, err := NormalizeEmail(" User@Example.com ")
	if err != nil || normalized != "user@example.com" {
		t.Fatalf("normalization failed: %s %v", normalized, err)
	}
	for _, endpoint := range []string{"javascript:alert(1)", "https://user:secret@example.com", "https://example.com/#token", "https://example.com/?q=test", "https:///path"} {
		settings := testSettings()
		settings.SiteURL = endpoint
		if err := settings.Validate(); err == nil {
			t.Fatalf("accepted invalid URL %q", endpoint)
		}
	}
	settings := testSettings()
	if err := settings.Validate(); err != nil {
		t.Fatal(err)
	}
	settings.Security = "invalid"
	if err := settings.Validate(); err == nil {
		t.Fatal("accepted invalid security")
	}
	settings = testSettings()
	settings.FromName = "sender\r\nBcc: victim"
	if err := settings.Validate(); err == nil {
		t.Fatal("accepted header injection")
	}
	settings = testSettings()
	settings.Username = "user"
	if err := settings.Validate(); err == nil {
		t.Fatal("accepted missing auth password")
	}
}

// Exercise SMTP envelope and UTF-8 MIME content against a local relay.
func TestSMTPDeliveryAndRequiredSTARTTLS(t *testing.T) {
	for _, security := range []string{"none", "starttls"} {
		t.Run(security, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			received := make(chan string, 1)
			errors := make(chan error, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					errors <- err
					return
				}
				defer conn.Close()
				reader := textproto.NewReader(bufio.NewReader(conn))
				fmt.Fprint(conn, "220 localhost SMTP\r\n")
				for {
					line, err := reader.ReadLine()
					if err != nil {
						errors <- nil
						return
					}
					switch {
					case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
						fmt.Fprint(conn, "250 localhost\r\n")
					case strings.HasPrefix(line, "MAIL FROM:<sender@example.test>"), strings.HasPrefix(line, "RCPT TO:<user@example.test>"):
						fmt.Fprint(conn, "250 OK\r\n")
					case line == "DATA":
						fmt.Fprint(conn, "354 send data\r\n")
						data, err := reader.ReadDotBytes()
						if err != nil {
							errors <- err
							return
						}
						received <- string(data)
						fmt.Fprint(conn, "250 accepted\r\n")
					case line == "QUIT":
						fmt.Fprint(conn, "221 bye\r\n")
						errors <- nil
						return
					default:
						errors <- fmt.Errorf("unexpected SMTP command %q", line)
						return
					}
				}
			}()
			_, port, _ := net.SplitHostPort(listener.Addr().String())
			settings := testSettings()
			settings.Host = "127.0.0.1"
			settings.Port, _ = strconv.Atoi(port)
			settings.Security = security
			err = (SMTPSender{}).Send(context.Background(), settings, "user@example.test", "确认邮箱", "请点击链接：\nhttps://ask.example.test/account/email/#token=abc")
			if security == "starttls" {
				if err == nil || !strings.Contains(err.Error(), "STARTTLS") {
					t.Fatalf("STARTTLS silently downgraded: %v", err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				message := <-received
				parsed, err := mail.ReadMessage(strings.NewReader(message))
				if err != nil {
					t.Fatal(err)
				}
				if parsed.Header.Get("Content-Type") != "text/plain; charset=UTF-8" || parsed.Header.Get("Content-Transfer-Encoding") != "quoted-printable" {
					t.Fatal("missing UTF-8 MIME headers")
				}
				if !strings.Contains(message, "From:") || !strings.Contains(message, "To: user@example.test") {
					t.Fatal("missing envelope headers")
				}
			}
			if err := <-errors; err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLoginAuthRequiresTLSAndHandlesChallenges(t *testing.T) {
	auth := &loginAuth{"username", "password", "smtp.example.test"}
	if _, _, err := auth.Start(&smtp.ServerInfo{Name: "smtp.example.test", TLS: false}); err == nil {
		t.Fatal("sent remote credentials without TLS")
	}
	for challenge, expected := range map[string]string{"Username:": "username", "Password:": "password"} {
		got, err := auth.Next([]byte(challenge), true)
		if err != nil || string(got) != expected {
			t.Fatalf("challenge failed: %v", err)
		}
	}
	if _, err := auth.Next([]byte("unexpected"), true); err == nil {
		t.Fatal("accepted unexpected LOGIN challenge")
	}
}
