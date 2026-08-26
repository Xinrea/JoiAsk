package bilibili

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func jsonClient(handler func(*http.Request) string) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(handler(request)))}, nil
	})}
}

func TestValidateAccountAndFollowers(t *testing.T) {
	httpClient := jsonClient(func(r *http.Request) string {
		if r.Header.Get("Cookie") != "SESSDATA=test" {
			t.Fatal("cookie header was not forwarded")
		}
		switch r.URL.Path {
		case "/x/web-interface/nav":
			return `{"code":0,"message":"0","data":{"isLogin":true,"mid":123}}`
		case "/x/relation/fans":
			if r.URL.Query().Get("vmid") != "123" || r.URL.Query().Get("ps") != "50" || r.URL.Query().Get("gaia_source") != "main_web" || r.URL.Query().Get("web_location") != "333.1387" {
				t.Fatalf("unexpected query: %s", r.URL.RawQuery)
			}
			if r.Header.Get("Origin") != "https://space.bilibili.com" || r.Header.Get("Referer") != "https://space.bilibili.com/123/relation/fans" || r.Header.Get("Sec-Fetch-Site") != "same-site" {
				t.Fatalf("browser headers were not forwarded: %+v", r.Header)
			}
			return `{"code":0,"message":"0","data":{"list":[{"mid":456,"mtime":1700000000,"uname":"tester","face":"https://example.test/avatar.jpg"}]}}`
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
			return ""
		}
	})
	client := &Client{BaseURL: "https://example.test", HTTPClient: httpClient}
	uid, err := client.ValidateAccount(context.Background(), "SESSDATA=test")
	if err != nil || uid != 123 {
		t.Fatalf("unexpected nav result: uid=%d err=%v", uid, err)
	}
	followers, err := client.Followers(context.Background(), uid, "SESSDATA=test", 50)
	if err != nil || len(followers) != 1 || followers[0].MID != 456 || followers[0].MTime != 1700000000 {
		t.Fatalf("unexpected followers result: %+v err=%v", followers, err)
	}
}

func TestProfile(t *testing.T) {
	client := &Client{BaseURL: "https://example.test", HTTPClient: jsonClient(func(r *http.Request) string {
		if r.URL.Path != "/space/123" || !strings.Contains(r.Header.Get("User-Agent"), "iPhone") {
			t.Fatalf("unexpected profile request: %s", r.URL.String())
		}
		return `<html><script>window.__INITIAL_STATE__={"space":{"info":{"mid":123,"name":"Bilibili User","face":"https://i1.hdslb.com/avatar.jpg"}}};(function(){})();</script></html>`
	})}
	profile, err := client.Profile(context.Background(), 123)
	if err != nil || profile.MID != 123 || profile.Name != "Bilibili User" || profile.Face == "" {
		t.Fatalf("unexpected profile result: %+v err=%v", profile, err)
	}
}

func TestClientRejectsBusinessAndMalformedResponses(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "business error", body: `{"code":-101,"message":"账号未登录","data":{}}`, want: "账号未登录"},
		{name: "malformed", body: `{`, want: "响应无效"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &Client{BaseURL: "https://example.test", HTTPClient: jsonClient(func(*http.Request) string { return test.body })}
			_, err := client.ValidateAccount(context.Background(), "secret-cookie")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q error, got %v", test.want, err)
			}
			if strings.Contains(err.Error(), "secret-cookie") {
				t.Fatal("error leaked cookie")
			}
		})
	}
}
