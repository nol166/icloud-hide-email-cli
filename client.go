package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	setupBase   = "https://setup.icloud.com/setup/ws/1/validate"
	fallbackURL = "https://p68-maildomainws.icloud.com"
	clientBuild = "2608Build39"
	userAgent   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15"
)

type Client struct {
	httpc    *http.Client
	cookie   string
	clientID string
	dsid     string
	base     string // premiummailsettings web service root
}

type Alias struct {
	Hme      string `json:"hme"`
	Label    string `json:"label"`
	Note     string `json:"note"`
	IsActive bool   `json:"isActive"`
	Created  int64  `json:"createTimestamp"`
}

// NewClient loads saved cookies and discovers the account's dsid + HME server.
func NewClient() (*Client, error) {
	cookie, err := loadCookies()
	if err != nil {
		return nil, fmt.Errorf("no iCloud session found — run `hidemail auth` first")
	}
	c := &Client{
		httpc:    &http.Client{Timeout: 30 * time.Second},
		cookie:   cookie,
		clientID: uuidV4(),
	}
	if err := c.discover(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) params() url.Values {
	v := url.Values{}
	v.Set("clientBuildNumber", clientBuild)
	v.Set("clientMasteringNumber", clientBuild)
	v.Set("clientId", c.clientID)
	if c.dsid != "" {
		v.Set("dsid", c.dsid)
	}
	return v
}

func (c *Client) setHeaders(req *http.Request, json bool) {
	req.Header.Set("Cookie", c.cookie)
	req.Header.Set("Origin", "https://www.icloud.com")
	req.Header.Set("Referer", "https://www.icloud.com/")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if json {
		req.Header.Set("Content-Type", "application/json")
	}
}

// discover hits the account validate endpoint to get dsid + the HME server URL.
func (c *Client) discover() error {
	req, _ := http.NewRequest("GET", setupBase+"?"+c.params().Encode(), nil)
	c.setHeaders(req, false)
	resp, err := c.httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("iCloud session invalid or expired (HTTP %d) — run `hidemail auth` to refresh", resp.StatusCode)
	}
	dsid, base, err := parseValidate(body)
	if err != nil {
		return err
	}
	c.dsid, c.base = dsid, base
	return nil
}

// parseValidate extracts dsid and the premiummailsettings service root.
func parseValidate(body []byte) (dsid, base string, err error) {
	var v struct {
		DsInfo struct {
			Dsid json.Number `json:"dsid"`
		} `json:"dsInfo"`
		Webservices struct {
			PremiumMail struct {
				URL    string `json:"url"`
				Status string `json:"status"`
			} `json:"premiummailsettings"`
		} `json:"webservices"`
	}
	if err := json.Unmarshal(body, &v); err != nil {
		return "", "", fmt.Errorf("unexpected iCloud response: %w", err)
	}
	base = strings.TrimSpace(v.Webservices.PremiumMail.URL)
	if base == "" {
		base = fallbackURL // ponytail: some responses omit it; hardcoded shard as last resort
	}
	return v.DsInfo.Dsid.String(), base, nil
}

func (c *Client) do(method, path string, reqBody any) ([]byte, error) {
	var r io.Reader
	if reqBody != nil {
		b, _ := json.Marshal(reqBody)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path+"?"+c.params().Encode(), r)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req, reqBody != nil)
	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, snippet(body))
	}
	if err := apiError(body); err != nil {
		return nil, err
	}
	return body, nil
}

// Generate returns a candidate Hide My Email address (not yet claimed).
func (c *Client) Generate() (string, error) {
	body, err := c.do("POST", "/v1/hme/generate", map[string]any{})
	if err != nil {
		return "", err
	}
	var v struct {
		Result struct {
			Hme string `json:"hme"`
		} `json:"result"`
	}
	json.Unmarshal(body, &v)
	if v.Result.Hme == "" {
		return "", fmt.Errorf("iCloud returned no address: %s", snippet(body))
	}
	return v.Result.Hme, nil
}

// Reserve claims a generated candidate with a label + note.
func (c *Client) Reserve(hme, label, note string) error {
	_, err := c.do("POST", "/v1/hme/reserve", map[string]any{
		"hme": hme, "label": label, "note": note,
	})
	return err
}

// List returns all Hide My Email aliases on the account.
func (c *Client) List() ([]Alias, error) {
	body, err := c.do("GET", "/v2/hme/list", nil)
	if err != nil {
		return nil, err
	}
	var v struct {
		Result struct {
			HmeEmails []Alias `json:"hmeEmails"`
		} `json:"result"`
	}
	json.Unmarshal(body, &v)
	return v.Result.HmeEmails, nil
}

// apiError surfaces Apple's {"success":false,...,"reason":".."} envelope.
func apiError(body []byte) error {
	var v struct {
		Success *bool  `json:"success"`
		Reason  string `json:"reason"`
		Error   any    `json:"error"`
	}
	if json.Unmarshal(body, &v) != nil {
		return nil // not the standard envelope; let caller parse what it needs
	}
	if v.Success != nil && !*v.Success {
		msg := v.Reason
		if msg == "" {
			msg = fmt.Sprintf("%v", v.Error)
		}
		return fmt.Errorf("iCloud error: %s", msg)
	}
	return nil
}

func uuidV4() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func snippet(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}
