package dreamhost

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientWithMinimalArgs(t *testing.T) {
	c, err := NewClient("test123", nil, "")
	assert.Nil(t, err, "expect NewClient err to be nil")
	assert.NotNil(t, c, "expect NewClient DNSClient not to be nil")
	if actual := c.BaseURL.String(); actual != dreamhostBaseUrl {
		t.Errorf("expected BaseURL to be %v, got %v", dreamhostBaseUrl, actual)
	}
}

func TestNewClientWithEmptyApiKey(t *testing.T) {
	c, err := NewClient("", nil, "")
	assert.Error(t, err, "expect NewClient err to be an error")
	assert.Nil(t, c, "expect NewClient DNSClient to be nil")
}

func TestNewClientWithInvalidUrl(t *testing.T) {
	c, err := NewClient("test123", nil, "\x7f")
	assert.Error(t, err, "expect NewClient err to be an error")
	assert.Nil(t, c, "expect NewClient DNSClient to be nil")
}

func TestCreateRecord(t *testing.T) {
	expectedCmd := "dns-add_record"
	apiKey := "apikey123"
	recordValue := DNSRecordValue{"example.com", "TXT", "testValue"}

	svr := mockHttpResponse(200, `{"result":"success","data":"record_added"}`, func(r *http.Request) {
		assert.Equal(t, agentString, r.UserAgent(), "expect user agent string to be correct")
		q := r.URL.Query()
		assert.Equal(t, apiKey, q.Get("key"), "expect request to have the desired API key")
		assert.Equal(t, expectedCmd, q.Get("cmd"), "expect request to have the desired command")
		assert.Equal(t, "json", q.Get("format"), "expect request to have the desired format")
		assert.Equal(t, recordValue.Name, q.Get("record"), "expect request to reference the desired record name")
		assert.Equal(t, recordValue.RecordType, q.Get("type"), "expect request to reference the desired record type")
		assert.Equal(t, recordValue.Value, q.Get("value"), "expect request to reference the desired value")
		assert.Falsef(t, q.Has("unique_id"), "expect unique_id to not be present. Got %v", q.Get("unique_id"))
	})
	defer svr.Close()

	c, err := NewClient(apiKey, nil, svr.URL)
	assert.Nil(t, err, "expect NewClient err to be nil")

	err = c.CreateRecord(recordValue, "")
	assert.Nil(t, err, "expect CreateRecord err to be nil")
}

func TestDeleteRecord(t *testing.T) {
	expectedCmd := "dns-remove_record"
	apiKey := "apikey123"
	recordValue := DNSRecordValue{"example.com", "TXT", "testValue"}

	svr := mockHttpResponse(200, `{"data":"record_removed","result":"success"}`, func(r *http.Request) {
		assert.Equal(t, agentString, r.UserAgent(), "expect user agent string to be correct")
		q := r.URL.Query()
		assert.Equal(t, apiKey, q.Get("key"), "expect request to have the desired API key")
		assert.Equal(t, expectedCmd, q.Get("cmd"), "expect request to have the desired command")
		assert.Equal(t, "json", q.Get("format"), "expect request to have the desired format")
		assert.Equal(t, recordValue.Name, q.Get("record"), "expect request to reference the desired record name")
		assert.Equal(t, recordValue.RecordType, q.Get("type"), "expect request to reference the desired record type")
		assert.Equal(t, recordValue.Value, q.Get("value"), "expect request to reference the desired value")
		assert.Falsef(t, q.Has("unique_id"), "expect unique_id to not be present. Got %v", q.Get("unique_id"))
	})
	defer svr.Close()

	c, err := NewClient(apiKey, nil, svr.URL)
	assert.Nil(t, err, "expect NewClient err to be nil")

	err = c.DeleteRecord(recordValue, "")
	assert.Nil(t, err, "expect DeleteRecord err to be nil")
}

func TestCreateRecordWithUniqueId(t *testing.T) {
	uniqueId := "unique123"

	svr := mockHttpResponse(200, `{"result":"success","data":"record_added"}`, func(r *http.Request) {
		q := r.URL.Query()
		if actual := q.Get("unique_id"); actual != uniqueId {
			t.Errorf("Expected cmd to be %v, got %v", uniqueId, actual)
		}
	})
	defer svr.Close()

	c, err := NewClient("apikey123", nil, svr.URL)
	assert.Nil(t, err, "expect NewClient err to be nil")

	err = c.CreateRecord(DNSRecordValue{"example.com", "TXT", "testValue"}, uniqueId)
	assert.Nil(t, err, "expect CreateRecord err to be nil")
}

func TestCreateRecordWithRepeatUniqueId(t *testing.T) {
	svr := mockHttpResponse(200, `{"data":"unique_id_already_used","result":"error"}`, nil)
	defer svr.Close()

	c, err := NewClient("apikey123", nil, svr.URL)
	assert.Nil(t, err, "expect NewClient err to be nil")

	err = c.CreateRecord(DNSRecordValue{"example.com", "TXT", "testValue"}, "unique123")
	assert.Nil(t, err, "expect CreateRecord err to be nil")
}

func TestCreateRecord500Error(t *testing.T) {
	expectedErrContent := "dreamhost API returned unexpected status code 500"

	// Provide a payload that looks successful, but send a 500 error code
	svr := mockHttpResponse(500, `{"result":"success","data":"record_added"}`, nil)
	defer svr.Close()

	c, err := NewClient("testApiKey", nil, svr.URL)
	assert.Nil(t, err, "expect NewClient err to be nil")

	err = c.CreateRecord(DNSRecordValue{"example.com", "TXT", "testValue"}, "")
	assert.ErrorContains(t, err, expectedErrContent, "expect CreateRecord to return expected error")
}

func TestCreateRecordInvalidResponse(t *testing.T) {
	expectedErrContent := "failed to parse response"

	// This won't parse as JSON
	svr := mockHttpResponse(200, "invalid", nil)
	defer svr.Close()

	c, err := NewClient("testApiKey", nil, svr.URL)
	assert.Nil(t, err, "expect NewClient err to be nil")

	err = c.CreateRecord(DNSRecordValue{"example.com", "TXT", "testValue"}, "")
	assert.ErrorContains(t, err, expectedErrContent, "expect CreateRecord to return expected error")
}

func TestCreateRecordConnectionError(t *testing.T) {
	expectedErrContent := "HTTP request failed"
	svr := mockHttpResponse(200, "invalid", nil)

	// Close the server before we make the test request so that the client TCP connection gets rejected
	svr.Close()

	c, err := NewClient("testApiKey", nil, svr.URL)
	assert.Nil(t, err, "expect NewClient err to be nil")

	err = c.CreateRecord(DNSRecordValue{"example.com", "TXT", "testValue"}, "")
	assert.ErrorContains(t, err, expectedErrContent, "expect CreateRecord to return expected error")
}

func TestCreateRecordErrorResponse(t *testing.T) {
	expectedErrContent := "dreamhost API returned non-successful result"

	// Provide a payload that looks successful, but send a 500 error code
	svr := mockHttpResponse(200, `{"result":"error","data":"record_already_exists_remove_first"}`, nil)
	defer svr.Close()

	c, err := NewClient("testApiKey", nil, svr.URL)
	assert.Nil(t, err, "expect NewClient err to be nil")

	err = c.CreateRecord(DNSRecordValue{"example.com", "TXT", "testValue"}, "")
	assert.ErrorContains(t, err, expectedErrContent, "expect CreateRecord to return expected error")
}

func TestCreateRecordReturnsErrorWhenInputsAreMissing(t *testing.T) {
	c, err := NewClient("test123", nil, "")
	assert.Nil(t, err, "expect NewClient err to be nil")

	cases := map[DNSRecordValue]string{
		DNSRecordValue{"", "TXT", "testValue"}:         "DNSRecordValue.Name must not be empty",
		DNSRecordValue{"example.com", "", "testValue"}: "DNSRecordValue.RecordType must not be empty",
		DNSRecordValue{"example.com", "TXT", ""}:       "DNSRecordValue.Value must not be empty",
	}

	for record, expectedError := range cases {
		err = c.CreateRecord(record, "abc123")
		assert.ErrorContains(t, err, expectedError, "expect CreateRecord to return expected error")
	}
}

func mockHttpResponse(status int, body string, validator func(*http.Request)) *httptest.Server {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if validator != nil {
			validator(r)
		}
		w.WriteHeader(status)
		_, err := fmt.Fprint(w, body)
		if err != nil {
			panic(err)
		}
	}))
	return svr
}
