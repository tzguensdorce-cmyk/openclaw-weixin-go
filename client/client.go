// Package client provides a pure Go client for Tencent iLink WeChat bot APIs.
//
// Protocol behavior is implemented with reference to Tencent/openclaw-weixin.
// Author: jtai团队（曾能混&tang先森） <jwhna1@gmil.com>
// Official Site: https://jtai.cc
package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

const (
	ilinkAppID               = "bot"
	ilinkClientVersion       = 131329
	qrBotType                = "3"
	DefaultBaseURL           = "https://ilinkai.weixin.qq.com"
	DefaultChannelVersion    = "2.1.1"
	DefaultGetUpdatesTimeout = 20000
	DefaultQRPollTimeout     = 35 * time.Second
	DefaultClientIDPrefix    = "weixin"
	MessageTypeUser          = 1
	MessageTypeBot           = 2
	MessageStateNew          = 0
	MessageStateGenerating   = 1
	MessageStateFinish       = 2
	TypingStatusTyping       = 1
	TypingStatusCancel       = 2
	ErrCodeSessionExpired    = -14
)

// Logger is an optional debug logger for raw request/response traces.
type Logger interface {
	Printf(format string, args ...any)
}

// Options controls client initialization.
type Options struct {
	BaseURL        string
	HTTPClient     *http.Client
	ClientIDPrefix string
	ChannelVersion string
	Logger         Logger
}

// Client is a thin HTTP client for iLink bot APIs.
type Client struct {
	httpClient     *http.Client
	baseURL        string
	clientIDPrefix string
	channelVersion string
	logger         Logger
}

// BaseInfo mirrors the plugin-side base_info object.
type BaseInfo struct {
	ChannelVersion string `json:"channel_version,omitempty"`
}

// WeixinMessage is the iLink message envelope.
type WeixinMessage struct {
	Seq          int64          `json:"seq,omitempty"`
	MessageID    int64          `json:"message_id,omitempty"`
	FromUserID   string         `json:"from_user_id,omitempty"`
	ToUserID     string         `json:"to_user_id,omitempty"`
	ClientID     string         `json:"client_id,omitempty"`
	CreateTimeMs int64          `json:"create_time_ms,omitempty"`
	SessionID    string         `json:"session_id,omitempty"`
	MessageType  int            `json:"message_type,omitempty"`
	MessageState int            `json:"message_state,omitempty"`
	ItemList     []*MessageItem `json:"item_list,omitempty"`
	ContextToken string         `json:"context_token,omitempty"`
}

// MessageItem is a single message item.
type MessageItem struct {
	Type     int       `json:"type"`
	TextItem *TextItem `json:"text_item,omitempty"`
}

// TextItem contains plain text content.
type TextItem struct {
	Text string `json:"text"`
}

// QRCodeResp is the login QR response.
type QRCodeResp struct {
	ErrCode int    `json:"ret"`
	ErrMsg  string `json:"errmsg"`
	QRCode  string `json:"qrcode"`
	QRUrl   string `json:"qrcode_img_content"`
}

// QRStatusResp is the QR polling response.
type QRStatusResp struct {
	ErrCode   int    `json:"ret"`
	ErrMsg    string `json:"errmsg"`
	Status    string `json:"status"`
	BotToken  string `json:"bot_token"`
	AccountID string `json:"account_id"`
	BaseURL   string `json:"baseurl"`
}

// GetUpdatesReq is the long-polling request.
type GetUpdatesReq struct {
	SyncBuf       string    `json:"get_updates_buf,omitempty"`
	LegacySyncBuf string    `json:"sync_buf,omitempty"`
	BaseInfo      *BaseInfo `json:"base_info,omitempty"`
}

// GetUpdatesResp is the long-polling response.
type GetUpdatesResp struct {
	Ret                  int              `json:"ret"`
	ErrCode              int              `json:"errcode"`
	ErrMsg               string           `json:"errmsg"`
	MessageList          []*WeixinMessage `json:"msgs,omitempty"`
	SyncBuf              string           `json:"get_updates_buf,omitempty"`
	LegacySyncBuf        string           `json:"sync_buf,omitempty"`
	LongPollingTimeoutMs int              `json:"longpolling_timeout_ms,omitempty"`
}

// SendMessageReq sends a generic message envelope.
type SendMessageReq struct {
	Msg      *WeixinMessage `json:"msg"`
	BaseInfo *BaseInfo      `json:"base_info,omitempty"`
}

// SendMessageResp is the send response.
type SendMessageResp struct {
	Ret     int    `json:"ret"`
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// SendTypingReq sends typing state.
type SendTypingReq struct {
	IlinkUserID string    `json:"ilink_user_id"`
	Ticket      string    `json:"typing_ticket,omitempty"`
	Status      int       `json:"status,omitempty"`
	BaseInfo    *BaseInfo `json:"base_info,omitempty"`
}

// New creates a new client with sane defaults.
func New(opts Options) *Client {
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	channelVersion := opts.ChannelVersion
	if channelVersion == "" {
		channelVersion = DefaultChannelVersion
	}
	clientIDPrefix := opts.ClientIDPrefix
	if clientIDPrefix == "" {
		clientIDPrefix = DefaultClientIDPrefix
	}
	return &Client{
		httpClient:     httpClient,
		baseURL:        baseURL,
		clientIDPrefix: clientIDPrefix,
		channelVersion: channelVersion,
		logger:         opts.Logger,
	}
}

// GenerateClientID creates a sendmessage client_id with a configurable prefix.
func GenerateClientID(prefix string) string {
	if prefix == "" {
		prefix = DefaultClientIDPrefix
	}
	return fmt.Sprintf("%s_%d_%06d", prefix, time.Now().UnixMilli(), rand.Intn(1000000))
}

// GenerateClientID creates a client_id using the client's prefix.
func (c *Client) GenerateClientID() string {
	return GenerateClientID(c.clientIDPrefix)
}

func (c *Client) buildBaseInfo() *BaseInfo {
	return &BaseInfo{ChannelVersion: c.channelVersion}
}

func (c *Client) buildHeaders(token string) http.Header {
	h := http.Header{}
	if token != "" {
		h.Set("AuthorizationType", "ilink_bot_token")
		h.Set("Authorization", "Bearer "+token)
	}
	uin := base64.StdEncoding.EncodeToString([]byte(strconv.FormatUint(uint64(rand.Uint32()), 10)))
	h.Set("X-WECHAT-UIN", uin)
	h.Set("iLink-App-Id", ilinkAppID)
	h.Set("iLink-App-ClientVersion", strconv.Itoa(ilinkClientVersion))
	h.Set("Content-Type", "application/json")
	return h
}

func (c *Client) logf(format string, args ...any) {
	if c.logger != nil {
		c.logger.Printf(format, args...)
	}
}

func (c *Client) doGet(ctx context.Context, url string, token string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("construct request: %w", err)
	}
	req.Header = c.buildHeaders(token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	c.logf("[ilink-get] url=%s status=%d raw=%s", url, resp.StatusCode, string(body))
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) doPost(ctx context.Context, url string, token string, body interface{}, out interface{}) error {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("construct request: %w", err)
	}
	req.Header = c.buildHeaders(token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	c.logf("[ilink-post] url=%s status=%d request=%s raw=%s", url, resp.StatusCode, string(bodyBytes), string(respBody))
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// FetchQRCode requests a login QR code from the iLink service.
func (c *Client) FetchQRCode(ctx context.Context) (*QRCodeResp, error) {
	url := c.baseURL + "/ilink/bot/get_bot_qrcode?bot_type=" + qrBotType
	var resp QRCodeResp
	if err := c.doGet(ctx, url, "", &resp); err != nil {
		return nil, err
	}
	if resp.ErrCode != 0 {
		return nil, fmt.Errorf("fetchQRCode API error %d: %s", resp.ErrCode, resp.ErrMsg)
	}
	return &resp, nil
}

// PollQRStatus checks the current login state for a QR code.
func (c *Client) PollQRStatus(ctx context.Context, qrCode string, baseURL string) (*QRStatusResp, error) {
	if baseURL == "" {
		baseURL = c.baseURL
	}
	url := fmt.Sprintf("%s/ilink/bot/get_qrcode_status?qrcode=%s", baseURL, qrCode)
	oldTimeout := c.httpClient.Timeout
	c.httpClient.Timeout = DefaultQRPollTimeout + 5*time.Second
	defer func() { c.httpClient.Timeout = oldTimeout }()

	var resp QRStatusResp
	if err := c.doGet(ctx, url, "", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// WaitLogin blocks until the QR login succeeds, is canceled, expires, or the context ends.
func (c *Client) WaitLogin(ctx context.Context, qrCode string, baseURL string, pollInterval time.Duration) (*QRStatusResp, error) {
	if pollInterval <= 0 {
		pollInterval = 1500 * time.Millisecond
	}
	for {
		resp, err := c.PollQRStatus(ctx, qrCode, baseURL)
		if err != nil {
			return nil, err
		}
		if resp.ErrCode == 0 && resp.BotToken != "" {
			return resp, nil
		}
		switch resp.Status {
		case "cancel", "cancelled":
			return resp, fmt.Errorf("login canceled by user")
		case "expired", "expire":
			return resp, fmt.Errorf("login QR expired")
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

// GetUpdates performs one long-poll request.
func (c *Client) GetUpdates(ctx context.Context, token string, syncBuf string, timeoutMS int) (*GetUpdatesResp, error) {
	if timeoutMS <= 0 {
		timeoutMS = DefaultGetUpdatesTimeout
	}
	url := c.baseURL + "/ilink/bot/getupdates"
	reqBody := GetUpdatesReq{
		SyncBuf:  syncBuf,
		BaseInfo: c.buildBaseInfo(),
	}

	oldTimeout := c.httpClient.Timeout
	c.httpClient.Timeout = time.Duration(timeoutMS/1000+10) * time.Second
	defer func() { c.httpClient.Timeout = oldTimeout }()

	var resp GetUpdatesResp
	if err := c.doPost(ctx, url, token, reqBody, &resp); err != nil {
		return nil, err
	}
	if resp.SyncBuf == "" {
		resp.SyncBuf = resp.LegacySyncBuf
	}
	if resp.ErrCode == 0 && resp.Ret != 0 {
		resp.ErrCode = resp.Ret
	}
	return &resp, nil
}

// SendMessage sends a fully constructed request body.
func (c *Client) SendMessage(ctx context.Context, token string, req *SendMessageReq) (*SendMessageResp, error) {
	url := c.baseURL + "/ilink/bot/sendmessage"
	if req != nil && req.BaseInfo == nil {
		req.BaseInfo = c.buildBaseInfo()
	}
	var resp SendMessageResp
	if err := c.doPost(ctx, url, token, req, &resp); err != nil {
		return nil, err
	}
	if resp.ErrCode == 0 && resp.Ret != 0 {
		resp.ErrCode = resp.Ret
	}
	if resp.ErrCode != 0 {
		return nil, fmt.Errorf("sendMessage API error %d: %s", resp.ErrCode, resp.ErrMsg)
	}
	return &resp, nil
}

// SendTextMessage is a convenience helper for plain text replies.
func (c *Client) SendTextMessage(ctx context.Context, token, toUserID, text, contextToken string) (*SendMessageResp, error) {
	return c.SendMessage(ctx, token, &SendMessageReq{
		Msg: &WeixinMessage{
			ToUserID:     toUserID,
			ClientID:     c.GenerateClientID(),
			MessageType:  MessageTypeBot,
			MessageState: MessageStateFinish,
			ContextToken: contextToken,
			ItemList: []*MessageItem{
				{
					Type:     1,
					TextItem: &TextItem{Text: text},
				},
			},
		},
	})
}

// SendTyping sends one typing-state request.
func (c *Client) SendTyping(ctx context.Context, token string, req *SendTypingReq) error {
	url := c.baseURL + "/ilink/bot/sendtyping"
	if req != nil && req.BaseInfo == nil {
		req.BaseInfo = c.buildBaseInfo()
	}
	var resp struct {
		Ret     int    `json:"ret"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := c.doPost(ctx, url, token, req, &resp); err != nil {
		return err
	}
	if resp.ErrCode == 0 && resp.Ret != 0 {
		resp.ErrCode = resp.Ret
	}
	if resp.ErrCode != 0 {
		return fmt.Errorf("sendTyping API error %d: %s", resp.ErrCode, resp.ErrMsg)
	}
	return nil
}

// SendTypingStatus is a convenience helper for typing indicators.
func (c *Client) SendTypingStatus(ctx context.Context, token, userID, ticket string, status int) error {
	return c.SendTyping(ctx, token, &SendTypingReq{
		IlinkUserID: userID,
		Ticket:      ticket,
		Status:      status,
	})
}
