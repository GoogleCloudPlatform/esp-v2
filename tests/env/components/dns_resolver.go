// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package components

import (
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"
)

type handler struct {
	records map[string]string
}

const healthCheckInterval = time.Millisecond * 200

func (h *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	switch r.Question[0].Qtype {
	case dns.TypeA:
		msg.Authoritative = true
		domain := msg.Question[0].Name

		if address, ok := h.records[domain]; ok {
			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
				A:   net.ParseIP(address),
			})
		}
	}
	_ = w.WriteMsg(&msg)
}

func NewDnsResolver(port uint16, records map[string]string) *dns.Server {
	return &dns.Server{
		Addr: fmt.Sprintf(":%v", port),
		Net:  "udp",
		Handler: &handler{
			records: records,
		},
	}
}

func QueryDnsResolver(dnsResolverAddress, target string) ([]*net.IP, error) {
	c := dns.Client{}
	m := dns.Msg{}
	m.SetQuestion(target+".", dns.TypeA)
	r, _, err := c.Exchange(&m, dnsResolverAddress)
	if err != nil {
		return nil, err
	}

	if len(r.Answer) == 0 {
		return nil, nil
	}

	var ret []*net.IP
	for _, ans := range r.Answer {
		Arecord := ans.(*dns.A)
		ret = append(ret, &Arecord.A)
	}

	return ret, nil
}

func CheckDnsResolverHealth(dnsResolverAddress, host, expectIp string) error {
	var ips []*net.IP
	var err error
	for i := 0; i < 10; i++ {
		ips, err = QueryDnsResolver(dnsResolverAddress, host)
		if err == nil {
			break
		}

		time.Sleep(healthCheckInterval)
	}

	if err != nil {
		return err
	}

	if len(ips) == 0 || ips[0].String() != expectIp {
		return fmt.Errorf("cannot get the dns records")
	}

	return nil
}
