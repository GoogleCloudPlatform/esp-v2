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
	"bytes"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func readPEM(t *testing.T, path, wantType string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != wantType {
		t.Fatalf("%s: want a %q PEM block, got %v", path, wantType, block)
	}
	return block.Bytes
}

func TestGenerateSelfSignedCert(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ssl", "endpoints")
	before := time.Now()
	if err := generateSelfSignedCert(dir); err != nil {
		t.Fatal(err)
	}

	cert, err := x509.ParseCertificate(readPEM(t, filepath.Join(dir, "server.crt"), "CERTIFICATE"))
	if err != nil {
		t.Fatal(err)
	}
	// What `openssl req -x509 -newkey rsa:2048 -nodes -days 3650
	// -subj /CN=localhost` produced, plus a localhost SAN.
	if cert.Subject.CommonName != "localhost" {
		t.Errorf("CN = %q, want localhost", cert.Subject.CommonName)
	}
	if len(cert.DNSNames) != 1 || cert.DNSNames[0] != "localhost" {
		t.Errorf("DNS SANs = %q, want [localhost]", cert.DNSNames)
	}
	if !cert.BasicConstraintsValid || !cert.IsCA {
		t.Errorf("want a CA:TRUE certificate like openssl's")
	}
	if len(cert.SubjectKeyId) == 0 || !bytes.Equal(cert.SubjectKeyId, cert.AuthorityKeyId) {
		t.Errorf("want matching subject and authority key IDs, got %x and %x", cert.SubjectKeyId, cert.AuthorityKeyId)
	}
	if !cert.NotBefore.Before(before.Add(-4 * time.Minute)) {
		t.Errorf("NotBefore = %v, want a few minutes before %v", cert.NotBefore, before)
	}
	if want := before.Add(3650 * 24 * time.Hour); cert.NotAfter.Before(want.Add(-time.Minute)) || cert.NotAfter.After(want.Add(time.Minute)) {
		t.Errorf("NotAfter = %v, want about %v", cert.NotAfter, want)
	}
	if pub, ok := cert.PublicKey.(*rsa.PublicKey); !ok || pub.N.BitLen() != 2048 {
		t.Errorf("want an RSA 2048 key, got %T", cert.PublicKey)
	}

	// The key is PKCS #8, like openssl 3 writes it, and private to the user.
	if _, err := x509.ParsePKCS8PrivateKey(readPEM(t, filepath.Join(dir, "server.key"), "PRIVATE KEY")); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(dir, "server.key")); err != nil {
		t.Error(err)
	} else if info.Mode().Perm() != 0600 {
		t.Errorf("server.key mode = %v, want 0600", info.Mode().Perm())
	}

	// A client that trusts the certificate can connect to "localhost".
	if _, err := tls.LoadX509KeyPair(filepath.Join(dir, "server.crt"), filepath.Join(dir, "server.key")); err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(cert)
	if _, err := cert.Verify(x509.VerifyOptions{DNSName: "localhost", Roots: roots}); err != nil {
		t.Errorf("certificate does not verify for localhost: %v", err)
	}
}

func TestGenProxyConfigSelfSignedCertDir(t *testing.T) {
	unsetEnvForTest(t, googleCredsEnv)
	l := newLauncher()
	l.certDir = filepath.Join(t.TempDir(), "certs")
	args, err := parseArgs([]string{"--version=v1", "--generate_self_signed_cert"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := l.genProxyConfig(args)
	if err != nil {
		t.Fatal(err)
	}
	// configmanager is pointed at the directory the files were written to.
	for i, arg := range got {
		if arg == "--ssl_server_cert_path" {
			if i+1 >= len(got) || got[i+1] != l.certDir {
				t.Fatalf("--ssl_server_cert_path is not %s in %q", l.certDir, got)
			}
			if _, err := os.Stat(filepath.Join(l.certDir, "server.crt")); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatalf("no --ssl_server_cert_path in %q", got)
}
