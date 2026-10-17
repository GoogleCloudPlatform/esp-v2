// Copyright 2026 Google LLC
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

package startproxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// defaultSelfSignedCertDir is the directory where self-signed certs are written.
const defaultSelfSignedCertDir = "/tmp/ssl/endpoints"

// generateSelfSignedCert writes a self-signed certificate for localhost and
// its private key to dir/server.crt and dir/server.key.
//
// The certificate matches what start_proxy.py generated with
//
//	openssl req -x509 -newkey rsa:2048 -nodes -days 3650 -subj /CN=localhost
//
// (a CA certificate with subject and authority key identifiers, and a PKCS #8
// key), plus a DNS:localhost subject alternative name, because modern TLS
// clients ignore the common name, and a NotBefore five minutes in the past to
// tolerate clock skew.
func generateSelfSignedCert(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generating key: %v", err)
	}
	// openssl uses a random, positive 160-bit serial number.
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 159))
	if err != nil {
		return fmt.Errorf("generating serial number: %v", err)
	}
	// Key identifier: SHA-1 of the public key (RFC 5280, section 4.2.1.2).
	keyID := sha1.Sum(x509.MarshalPKCS1PublicKey(&key.PublicKey))

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "localhost"},
		DNSNames:              []string{"localhost"},
		NotBefore:             now.Add(-5 * time.Minute),
		NotAfter:              now.Add(3650 * 24 * time.Hour),
		BasicConstraintsValid: true,
		IsCA:                  true,
		SubjectKeyId:          keyID[:],
		AuthorityKeyId:        keyID[:],
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("creating certificate: %v", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return fmt.Errorf("encoding key: %v", err)
	}
	if err := writePEM(filepath.Join(dir, "server.crt"), "CERTIFICATE", certDER, 0644); err != nil {
		return err
	}
	return writePEM(filepath.Join(dir, "server.key"), "PRIVATE KEY", keyDER, 0600)
}

func writePEM(path, blockType string, der []byte, perm os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if err := pem.Encode(f, &pem.Block{Type: blockType, Bytes: der}); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
