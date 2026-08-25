package bilibili

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const defaultBaseURL = "https://api.bilibili.com"

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

type Follower struct {
	MID   int64  `json:"mid"`
	MTime int64  `json:"mtime"`
	Name  string `json:"uname"`
	Face  string `json:"face"`
}

type Profile struct {
	MID       int64  `json:"mid"`
	Name      string `json:"name"`
	Face      string `json:"face"`
	Signature string `json:"sign"`
}

type apiResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type navData struct {
	IsLogin bool  `json:"isLogin"`
	MID     int64 `json:"mid"`
}

type followerData struct {
	List []Follower `json:"list"`
}

func NewClient() *Client {
	return &Client{BaseURL: defaultBaseURL, HTTPClient: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Client) endpoint(path string) string {
	if c.BaseURL == "" {
		return defaultBaseURL + path
	}
	return c.BaseURL + path
}

func (c *Client) do(ctx context.Context, endpoint, cookie string, target any) error {
	return c.doWithHeaders(ctx, endpoint, cookie, target, nil)
}

func (c *Client) doWithHeaders(ctx context.Context, endpoint, cookie string, target any, extra http.Header) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en,zh;q=0.9,zh-CN;q=0.8")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Priority", "u=1, i")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/151.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.bilibili.com/")
	for key, values := range extra {
		if len(values) > 0 {
			req.Header.Set(key, values[0])
		}
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("B 站请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("B 站返回 HTTP %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("B 站响应无效: %w", err)
	}
	return nil
}

func (c *Client) ValidateAccount(ctx context.Context, cookie string) (int64, error) {
	var response apiResponse[navData]
	if err := c.do(ctx, c.endpoint("/x/web-interface/nav"), cookie, &response); err != nil {
		return 0, err
	}
	if response.Code != 0 {
		return 0, fmt.Errorf("B 站登录校验失败: %s (%d)", response.Message, response.Code)
	}
	if !response.Data.IsLogin || response.Data.MID <= 0 {
		return 0, errors.New("B 站 Cookie 已失效或未登录")
	}
	return response.Data.MID, nil
}

func (c *Client) Followers(ctx context.Context, uid int64, cookie string, limit int) ([]Follower, error) {
	if limit < 1 || limit > 50 {
		limit = 50
	}
	values := url.Values{}
	values.Set("vmid", strconv.FormatInt(uid, 10))
	values.Set("pn", "1")
	values.Set("ps", strconv.Itoa(limit))
	values.Set("gaia_source", "main_web")
	values.Set("web_location", "333.1387")
	var response apiResponse[followerData]
	headers := http.Header{}
	headers.Set("Origin", "https://space.bilibili.com")
	headers.Set("Referer", "https://space.bilibili.com/"+strconv.FormatInt(uid, 10)+"/relation/fans")
	headers.Set("Sec-CH-UA", `"Not=A?Brand";v="99", "Google Chrome";v="151", "Chromium";v="151"`)
	headers.Set("Sec-CH-UA-Mobile", "?0")
	headers.Set("Sec-CH-UA-Platform", `"macOS"`)
	headers.Set("Sec-Fetch-Dest", "empty")
	headers.Set("Sec-Fetch-Mode", "cors")
	headers.Set("Sec-Fetch-Site", "same-site")
	if err := c.doWithHeaders(ctx, c.endpoint("/x/relation/fans")+"?"+values.Encode(), cookie, &response, headers); err != nil {
		return nil, err
	}
	if response.Code != 0 {
		return nil, fmt.Errorf("获取 B 站粉丝失败: %s (%d)", response.Message, response.Code)
	}
	return response.Data.List, nil
}

func (c *Client) Profile(ctx context.Context, uid int64) (Profile, error) {
	values := url.Values{}
	values.Set("mid", strconv.FormatInt(uid, 10))
	var response apiResponse[Profile]
	if err := c.do(ctx, c.endpoint("/x/space/acc/info")+"?"+values.Encode(), "", &response); err != nil {
		return Profile{}, err
	}
	if response.Code != 0 {
		return Profile{}, fmt.Errorf("获取 B 站用户信息失败: %s (%d)", response.Message, response.Code)
	}
	if response.Data.MID != uid || response.Data.Name == "" {
		return Profile{}, errors.New("B 站用户信息无效")
	}
	return response.Data, nil
}
