package main

import (
	"fmt"
	"github.com/nprzy/cert-manager-webhook-dreamhost/internal/solvertest"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	acmetest "github.com/cert-manager/cert-manager/test/acme"
)

var (
	zone      = os.Getenv("TEST_ZONE_NAME")
	dnsServer = os.Getenv("TEST_DNS_SERVER")
	apiUrl    = os.Getenv("TEST_API_URL")
)

// This test launches a mock HTTP server to emulate the DreamHost API and a local DNS server. Calls to the
// mock DreamHost API will result in the creation or deletion of local DNS records.
// This local DNS server logic has a minor caveat. See internal/solvertest/dns.go for details.
func TestRunsSuite(t *testing.T) {
	dnsPort := "59351"
	expectedKey := "DREAMHOST_API_KEY_TESTDATA" // value in testdata/dreamhost-solver/secret.yaml

	dnssvr := solvertest.NewTestDnsServer(dnsPort)

	apisvr := mockHttpResponse(200, `{"result":"success","data":"record_added"}`, func(r *http.Request) {
		q := r.URL.Query()
		if actual := q.Get("key"); actual != expectedKey {
			t.Errorf("Expected key to be %v, got %v", expectedKey, actual)
		}
		if actual := q.Get("type"); actual != "TXT" {
			t.Errorf("Expected type to be TXT, got %v", actual)
		}

		cmd := q.Get("cmd")
		record := q.Get("record")
		value := q.Get("value")
		if cmd == "dns-add_record" {
			dnssvr.AddTxtRecord(record, value)
		} else if cmd == "dns-remove_record" {
			dnssvr.DeleteTxtRecord(record)
		} else {
			t.Errorf("API received unexpected cmd %v", cmd)
		}
	})
	defer apisvr.Close()
	defer dnssvr.Shutdown()

	// By default we run a local unit test with mocked API/DNS endpoints. But if the environment variables are
	// configured appropriately, we can run against real DreamHost instead.
	if zone == "" {
		zone = "example.com."
	}
	if dnsServer == "" && apiUrl == "" {
		fmt.Print("Using local API/DNS mocks\n")
		dnsServer = "127.0.0.1:" + dnsPort
		apiUrl = apisvr.URL
	}

	fixture := acmetest.NewFixture(&dreamHostDnsProviderSolver{baseUrl: apiUrl},
		acmetest.SetResolvedZone(zone),
		acmetest.SetAllowAmbientCredentials(false),
		acmetest.SetManifestPath("testdata/dreamhost-solver"),
		acmetest.SetDNSServer(dnsServer),
		acmetest.SetUseAuthoritative(false),
	)
	//need to uncomment and  RunConformance delete runBasic and runExtended once https://github.com/cert-manager/cert-manager/pull/4835 is merged
	//fixture.RunConformance(t)
	fixture.RunBasic(t)
	fixture.RunExtended(t)

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
