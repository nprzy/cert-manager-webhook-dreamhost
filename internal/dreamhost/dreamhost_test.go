package dreamhost

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClientWithMinimalArgs(t *testing.T) {
	c, err := NewClient("test123", nil, "")
	if err != nil {
		t.Errorf("expected NewClient err to be nil, got %v", err)
	}
	if c == nil {
		t.Error("expected NewClient DNSClient not to be nil, got nil")
	}
	if actual := c.BaseURL.String(); actual != dreamhostBaseUrl {
		t.Errorf("expected BaseURL to be %v, got %v", dreamhostBaseUrl, actual)
	}
}

func TestNewClientWithEmptyApiKey(t *testing.T) {
	c, err := NewClient("", nil, "")
	if err == nil {
		t.Error("expected NewClient to return err, got nil")
	}
	if c != nil {
		t.Error("expected NewClient DNSClient to be nil, was not nil")
	}
}

func TestNewClientWithInvalidUrl(t *testing.T) {
	c, err := NewClient("test123", nil, "\x7f")
	if err == nil {
		t.Error("expected NewClient to return err, got nil")
	}
	if c != nil {
		t.Error("expected NewClient DNSClient to be nil, was not nil")
	}
}

func TestCreateRecord(t *testing.T) {
	expectedCmd := "dns-add_record"
	apiKey := "apikey123"
	recordValue := DNSRecordValue{"example.com", "TXT", "testValue"}

	svr := mockHttpResponse(200, `{"result":"success","data":"record_added"}`, func(r *http.Request) {
		if r.UserAgent() != agentString {
			t.Errorf("Expected user agent to be %v, got %v", agentString, r.URL.Scheme)
		}
		q := r.URL.Query()
		if actual := q.Get("key"); actual != apiKey {
			t.Errorf("Expected key to be %v, got %v", apiKey, actual)
		}
		if actual := q.Get("cmd"); actual != expectedCmd {
			t.Errorf("Expected cmd to be %v, got %v", expectedCmd, actual)
		}
		if actual := q.Get("format"); actual != "json" {
			t.Errorf("Expected format to be json, got %v", actual)
		}
		if actual := q.Get("record"); actual != recordValue.Name {
			t.Errorf("Expected record to be %v, got %v", recordValue.Name, actual)
		}
		if actual := q.Get("type"); actual != recordValue.RecordType {
			t.Errorf("Expected type to be %v, got %v", recordValue.RecordType, actual)
		}
		if actual := q.Get("value"); actual != recordValue.Value {
			t.Errorf("Expected value to be %v, got %v", recordValue.Value, actual)
		}
		if q.Has("unique_id") {
			t.Errorf("Expected unique_id to not be present, got %v", q.Get("unique_id"))
		}
	})
	defer svr.Close()

	c, err := NewClient(apiKey, nil, svr.URL)
	if err != nil {
		t.Errorf("expected NewClient err to be nil, got %v", err)
	}

	err = c.CreateRecord(recordValue, "")
	if err != nil {
		t.Errorf("Expected CreateRecord not to return error, got %v", err)
	}
}

func TestDeleteRecord(t *testing.T) {
	expectedCmd := "dns-remove_record"
	apiKey := "apikey123"
	recordValue := DNSRecordValue{"example.com", "TXT", "testValue"}

	svr := mockHttpResponse(200, `{"data":"record_removed","result":"success"}`, func(r *http.Request) {
		if r.UserAgent() != agentString {
			t.Errorf("Expected user agent to be %v, got %v", agentString, r.URL.Scheme)
		}
		q := r.URL.Query()
		if actual := q.Get("key"); actual != apiKey {
			t.Errorf("Expected key to be %v, got %v", apiKey, actual)
		}
		if actual := q.Get("cmd"); actual != expectedCmd {
			t.Errorf("Expected cmd to be %v, got %v", expectedCmd, actual)
		}
		if actual := q.Get("format"); actual != "json" {
			t.Errorf("Expected format to be json, got %v", actual)
		}
		if actual := q.Get("record"); actual != recordValue.Name {
			t.Errorf("Expected record to be %v, got %v", recordValue.Name, actual)
		}
		if actual := q.Get("type"); actual != recordValue.RecordType {
			t.Errorf("Expected type to be %v, got %v", recordValue.RecordType, actual)
		}
		if actual := q.Get("value"); actual != recordValue.Value {
			t.Errorf("Expected value to be %v, got %v", recordValue.Value, actual)
		}
		if q.Has("unique_id") {
			t.Errorf("Expected unique_id to not be present, got %v", q.Get("unique_id"))
		}
	})
	defer svr.Close()

	c, err := NewClient(apiKey, nil, svr.URL)
	if err != nil {
		t.Errorf("expected NewClient err to be nil, got %v", err)
	}

	err = c.DeleteRecord(recordValue, "")
	if err != nil {
		t.Errorf("Expected DeleteRecord not to return error, got %v", err)
	}
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
	if err != nil {
		t.Errorf("expected NewClient err to be nil, got %v", err)
	}

	err = c.CreateRecord(DNSRecordValue{"example.com", "TXT", "testValue"}, uniqueId)
	if err != nil {
		t.Errorf("Expected CreateRecord not to return error, got %v", err)
	}
}

func TestCreateRecordWithRepeatUniqueId(t *testing.T) {
	svr := mockHttpResponse(200, `{"data":"unique_id_already_used","result":"error"}`, nil)
	defer svr.Close()

	c, _ := NewClient("apikey123", nil, svr.URL)
	if err := c.CreateRecord(DNSRecordValue{"example.com", "TXT", "testValue"}, "unique123"); err != nil {
		t.Errorf("Expected CreateRecord not to return error, got %v", err)
	}
}

func TestCreateRecord500Error(t *testing.T) {
	expectedErrContent := "dreamhost API returned unexpected status code 500"

	// Provide a payload that looks successful, but send a 500 error code
	svr := mockHttpResponse(500, `{"result":"success","data":"record_added"}`, nil)
	defer svr.Close()

	c, _ := NewClient("testApiKey", nil, svr.URL)
	if err := c.CreateRecord(DNSRecordValue{"example.com", "TXT", "testValue"}, ""); err == nil {
		t.Error("Expected CreateRecord to return error, got nil")
	} else if !strings.Contains(err.Error(), expectedErrContent) {
		t.Errorf("Expected err to contain %v, but was %v instead", expectedErrContent, err.Error())
	}
}

func TestCreateRecordInvalidResponse(t *testing.T) {
	expectedErrContent := "failed to parse response"

	// This won't parse as JSON
	svr := mockHttpResponse(200, "invalid", nil)
	defer svr.Close()

	c, _ := NewClient("testApiKey", nil, svr.URL)
	if err := c.CreateRecord(DNSRecordValue{"example.com", "TXT", "testValue"}, ""); err == nil {
		t.Error("Expected CreateRecord to return error, got nil")
	} else if !strings.Contains(err.Error(), expectedErrContent) {
		t.Errorf("Expected err to contain %v, but was %v instead", expectedErrContent, err.Error())
	}
}

func TestCreateRecordConnectionError(t *testing.T) {
	expectedErrContent := "HTTP request failed"
	svr := mockHttpResponse(200, "invalid", nil)

	// Close the server before we make the test request so that the client TCP connection gets rejected
	svr.Close()

	c, _ := NewClient("testApiKey", nil, svr.URL)
	if err := c.CreateRecord(DNSRecordValue{"example.com", "TXT", "testValue"}, ""); err == nil {
		t.Error("Expected CreateRecord to return error, got nil")
	} else if !strings.Contains(err.Error(), expectedErrContent) {
		t.Errorf("Expected err to contain %v, but was %v instead", expectedErrContent, err.Error())
	}
}

func TestCreateRecordErrorResponse(t *testing.T) {
	expectedErrContent := "dreamhost API returned non-successful result"

	// Provide a payload that looks successful, but send a 500 error code
	svr := mockHttpResponse(200, `{"result":"error","data":"record_already_exists_remove_first"}`, nil)
	defer svr.Close()

	c, _ := NewClient("testApiKey", nil, svr.URL)
	if err := c.CreateRecord(DNSRecordValue{"example.com", "TXT", "testValue"}, ""); err == nil {
		t.Error("Expected CreateRecord to return error, got nil")
	} else if !strings.Contains(err.Error(), expectedErrContent) {
		t.Errorf("Expected err to contain %v, but was %v instead", expectedErrContent, err.Error())
	}
}

func TestCreateRecordReturnsErrorWhenInputsAreMissing(t *testing.T) {
	c, err := NewClient("test123", nil, "")
	if err != nil {
		t.Errorf("expected NewClient err to be nil, got %v", err)
	}

	cases := map[DNSRecordValue]string{
		DNSRecordValue{"", "TXT", "testValue"}:         "DNSRecordValue.Name must not be empty",
		DNSRecordValue{"example.com", "", "testValue"}: "DNSRecordValue.RecordType must not be empty",
		DNSRecordValue{"example.com", "TXT", ""}:       "DNSRecordValue.Value must not be empty",
	}

	for record, expectedError := range cases {
		err = c.CreateRecord(record, "abc123")
		if err == nil || !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected CreateRecord to return error `%v`, but it was %v instead", expectedError, err)
		}
	}
}

func mockHttpResponse(status int, body string, validator func(*http.Request)) *httptest.Server {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if validator != nil {
			validator(r)
		}
		w.WriteHeader(status)
		_, err := fmt.Fprintf(w, body)
		if err != nil {
			panic(err)
		}
	}))
	return svr
}
