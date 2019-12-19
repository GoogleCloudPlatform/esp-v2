// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package components

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"time"
)

var (
	serverIPs = []string{"127.0.0.1"}

	// Default 1 year certificate expiration.
	defaultDuration = 365 * 24 * time.Hour
)

// GenerateCert generates a certificate from the root cert and key pair.
func GenerateCert(certPath, keyPath string) (*tls.Certificate, error) {
	rootCert, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cert file: %v", err)
	}
	key, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %v", err)
	}

	data, err := tls.X509KeyPair(rootCert, key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode X.509 keypair: %v", err)
	}

	if len(data.Certificate) != 1 {
		return nil, errors.New("invalid cert file: contains more than 1 certificate in chain")
	}
	cacert, err := x509.ParseCertificate(data.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %v", err)
	}

	addrs := []net.IP{}
	for _, ip := range serverIPs {
		addr := net.ParseIP(ip)
		if addr == nil {
			return nil, fmt.Errorf("invalid IP: %s", ip)
		}
		addrs = append(addrs, addr)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %s", err)
	}

	template := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               cacert.Subject,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(defaultDuration),
		IPAddresses:           addrs,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, cacert, cacert.PublicKey, data.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cert: %v", err)
	}

	var out bytes.Buffer
	if err := pem.Encode(&out, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		return nil, fmt.Errorf("failed to encode child certificate: %v", err)
	}

	cert, err := tls.X509KeyPair(out.Bytes(), key)
	if err != nil {
		return nil, fmt.Errorf("failed to load X.509 key pair for child certificate: %v", err)
	}
	return &cert, nil
}
