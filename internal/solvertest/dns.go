package solvertest

import (
	"fmt"
	"github.com/miekg/dns"
	"os"
	"sync"
)

// This implements a basic local DNS server for use in the unit tests. This code is mostly borrowed from
// https://github.com/cert-manager/webhook-example/tree/0dcb6537405096ec6415b5e81daa6568323e9833/example
// This implementation has a caveat that will make it behave a bit unlike it will in the real world. This
// only supports a single TXT value per DNS record. When adding a new TXT record, it will overwrite the old
// value. When deleting a TXT record, it will delete it regardless of whether the requested value to delete
// actually matches. The standard cert-manager webhook unit tests don't require this functionality so we
// can get by without it for now.

type TestDnsServer struct {
	server     *dns.Server
	txtRecords map[string]string
	sync.RWMutex
}

func (e *TestDnsServer) AddTxtRecord(fqdn string, value string) {
	e.Lock()
	e.txtRecords[fqdn+"."] = value
	e.Unlock()
}

func (e *TestDnsServer) DeleteTxtRecord(fqdn string) {
	e.Lock()
	delete(e.txtRecords, fqdn+".")
	e.Unlock()
}

func NewTestDnsServer(port string) *TestDnsServer {
	e := &TestDnsServer{
		txtRecords: make(map[string]string),
	}
	e.server = &dns.Server{
		Addr:    ":" + port,
		Net:     "udp",
		Handler: dns.HandlerFunc(e.handleDNSRequest),
	}

	go func() {
		fmt.Print("Starting test DNS server on port " + port + "\n")
		if err := e.server.ListenAndServe(); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			os.Exit(1)
		}
		fmt.Print("Test DNS server has shut down\n")
	}()

	return e
}

func (e *TestDnsServer) Shutdown() {
	if err := e.server.Shutdown(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
	}
}

func (e *TestDnsServer) handleDNSRequest(w dns.ResponseWriter, req *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(req)
	switch req.Opcode {
	case dns.OpcodeQuery:
		for _, q := range msg.Question {
			if err := e.addDNSAnswer(q, msg, req); err != nil {
				msg.SetRcode(req, dns.RcodeServerFailure)
				break
			}
		}
	}
	w.WriteMsg(msg)
}

func (e *TestDnsServer) addDNSAnswer(q dns.Question, msg *dns.Msg, req *dns.Msg) error {
	switch q.Qtype {
	// Always return loopback for any A query
	case dns.TypeA:
		rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN A 127.0.0.1", q.Name))
		if err != nil {
			return err
		}
		msg.Answer = append(msg.Answer, rr)
		return nil

	// TXT records are the only important record for ACME dns-01 challenges
	case dns.TypeTXT:
		e.RLock()
		record, found := e.txtRecords[q.Name]
		e.RUnlock()
		if !found {
			msg.SetRcode(req, dns.RcodeNameError)
			return nil
		}
		rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN TXT %s", q.Name, record))
		if err != nil {
			return err
		}
		msg.Answer = append(msg.Answer, rr)
		return nil

	// NS and SOA are for authoritative lookups, return obviously invalid data
	case dns.TypeNS:
		rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN NS ns.example-acme-webook.invalid.", q.Name))
		if err != nil {
			return err
		}
		msg.Answer = append(msg.Answer, rr)
		return nil
	case dns.TypeSOA:
		rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN SOA %s 20 5 5 5 5", "ns.example-acme-webook.invalid.", "ns.example-acme-webook.invalid."))
		if err != nil {
			return err
		}
		msg.Answer = append(msg.Answer, rr)
		return nil
	default:
		return fmt.Errorf("unimplemented record type %v", q.Qtype)
	}
}
