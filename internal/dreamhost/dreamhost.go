package dreamhost

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const agentString = "cert-manager-webhook-dreamhost/0.1"
const dreamhostBaseUrl = "https://api.dreamhost.com/"

// DNSClient is a client for creating and deleting DNS records using the Dreamhost DNS API.
//
// References:
//   - https://help.dreamhost.com/hc/en-us/articles/4407354972692-Connecting-to-the-DreamHost-API
//   - https://help.dreamhost.com/hc/en-us/articles/217555707-DNS-API-commands
type DNSClient struct {
	apiKey  string
	client  *http.Client
	BaseURL *url.URL
}

func NewClient(apiKey string, httpClient *http.Client, baseUrl string) (*DNSClient, error) {
	if apiKey == "" {
		return nil, errors.New("empty apiKey")
	}
	if httpClient == nil {
		httpClient = &http.Client{
			// There is no timeout by default.
			Timeout: time.Second * 15,
		}
	}
	if baseUrl == "" {
		baseUrl = dreamhostBaseUrl
	}

	apiUrl, err := url.Parse(baseUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	return &DNSClient{apiKey, httpClient, apiUrl}, nil
}

func (c *DNSClient) prepareRequest(req *http.Request, cmd string, uniqueId string) {
	req.Header.Set("User-Agent", agentString)

	q := req.URL.Query()
	q.Add("key", c.apiKey)
	q.Add("cmd", cmd)
	q.Add("format", "json")
	if uniqueId != "" {
		q.Add("unique_id", uniqueId)
	}
	req.URL.RawQuery = q.Encode()
}

// CreateRecord creates a DNS record. A uniqueId string may optionally be provided for idempotency.
//
// Example GET request:
// https://api.dreamhost.com/?key=1A2B3C4D5E6F7G8H&cmd=dns-add_record&record=example.com&type=TXT&value=test123&format=json&unique_id=123456
func (c *DNSClient) CreateRecord(r DNSRecordValue, uniqueId string) error {
	resp, err := c.sendRequest(&r, "dns-add_record", uniqueId)
	return suppressUniqueIdUsedErr(resp, err)
}

// DeleteRecord deletes a DNS record. A uniqueId string may optionally be provided for idempotency.
//
// Example GET request:
// https://api.dreamhost.com/?key=1A2B3C4D5E6F7G8H&cmd=dns-remove_record&record=example.com&type=TXT&value=test123&format=json&unique_id=123456
func (c *DNSClient) DeleteRecord(r DNSRecordValue, uniqueId string) error {
	resp, err := c.sendRequest(&r, "dns-remove_record", uniqueId)
	return suppressUniqueIdUsedErr(resp, err)
}

func (c *DNSClient) sendRequest(r *DNSRecordValue, cmd string, uniqueId string) (*DreamhostResponse, error) {
	apiUrl := c.BaseURL.String()

	// The URL needs to end with a trailing slash
	if !strings.HasSuffix(apiUrl, "/") {
		apiUrl += "/"
	}

	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.prepareRequest(req, cmd, uniqueId)
	if err := r.addToReq(req); err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	// The Dreamhost API seems to return a 200 status code, even when the response is an error.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("dreamhost API returned unexpected status code %v", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTTP body: %w", err)
	}

	var apiResp DreamhostResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Result != "success" {
		return &apiResp, fmt.Errorf("dreamhost API returned non-successful result: %v", apiResp)
	}

	return &apiResp, nil
}

func suppressUniqueIdUsedErr(resp *DreamhostResponse, err error) error {
	// If the reason for the error is "unique_id_already_used", suppress the error because we assume that the caller's
	// intent has been successfully fulfilled, albeit in a previous request.
	if err != nil && resp != nil && resp.Data == "unique_id_already_used" {
		return nil
	}
	return err
}

// DNSRecordValue represents a single record name/value pair.
type DNSRecordValue struct {
	Name       string
	RecordType string
	Value      string
}

func (r *DNSRecordValue) addToReq(req *http.Request) error {
	if r.Name == "" {
		return errors.New("DNSRecordValue.Name must not be empty")
	}
	// Allowing the DreamHost API to validate that the caller is requesting a valid RecordType.
	if r.RecordType == "" {
		return errors.New("DNSRecordValue.RecordType must not be empty")
	}
	if r.Value == "" {
		return errors.New("DNSRecordValue.Value must not be empty")
	}

	q := req.URL.Query()
	q.Add("record", r.Name)
	q.Add("type", r.RecordType)
	q.Add("value", r.Value)
	req.URL.RawQuery = q.Encode()
	return nil
}

type DreamhostResponse struct {
	Result string
	Data   string
	Reason string
}
