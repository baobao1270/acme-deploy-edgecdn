package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCaller(t *testing.T) {
	tests := []struct {
		input Caller
		want  Caller
		ok    bool
	}{
		{input: "", want: CallerCLI, ok: true},
		{input: CallerAcmeSH, want: CallerAcmeSH, ok: true},
		{input: CallerLego, want: CallerLego, ok: true},
		{input: CallerCLI, want: CallerCLI, ok: true},
		{input: "unknown", ok: false},
	}

	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			got, err := parseCaller(test.input)
			if (err == nil) != test.ok {
				t.Fatalf("parseCaller(%q) error = %v, want success=%v", test.input, err, test.ok)
			}
			if got != test.want {
				t.Fatalf("parseCaller(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestConfigValidateDuplicateProfiles(t *testing.T) {
	cfg := &Config{Profiles: []Profile{
		{Type: "esa", Domain: "Example.COM"},
		{Type: "esa", Domain: "example.com."},
	}}

	warnings, err := cfg.Validate(false)
	if err == nil {
		t.Fatal("Validate() error = nil, want duplicate profile error")
	}
	if len(warnings) != 1 {
		t.Fatalf("Validate() warnings = %d, want 1", len(warnings))
	}

	warnings, err = cfg.Validate(true)
	if err != nil {
		t.Fatalf("Validate(true) error = %v, want nil", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("Validate(true) warnings = %d, want 1", len(warnings))
	}
}

func TestResolveDomain(t *testing.T) {
	t.Setenv("Le_Domain", "env.example.com")
	t.Setenv("LEGO_HOOK_CERT_NAME", "")

	domain, err := resolveDomain(CallerAcmeSH, "")
	if err != nil {
		t.Fatalf("resolveDomain() error = %v", err)
	}
	if domain != "env.example.com" {
		t.Fatalf("resolveDomain() = %q, want acme.sh environment value", domain)
	}

	domain, err = resolveDomain(CallerAcmeSH, "flag.example.com")
	if err != nil {
		t.Fatalf("resolveDomain() with flag error = %v", err)
	}
	if domain != "flag.example.com" {
		t.Fatalf("resolveDomain() = %q, want flag value", domain)
	}

	if _, err := resolveDomain(CallerCLI, ""); err == nil {
		t.Fatal("resolveDomain(cli) error = nil, want missing domain error")
	}
	if domain, err := resolveDomain(CallerLego, ""); err != nil || domain != "" {
		t.Fatalf("resolveDomain(lego) = %q, %v; want an empty, skippable domain", domain, err)
	}
}

func TestResolveCertificatePaths(t *testing.T) {
	t.Setenv("CERT_FULLCHAIN_PATH", "/env/fullchain.pem")
	t.Setenv("CERT_KEY_PATH", "/env/key.pem")

	certPath, keyPath, err := resolveCertificatePaths(CallerAcmeSH, "", "")
	if err != nil {
		t.Fatalf("resolveCertificatePaths() error = %v", err)
	}
	if certPath != "/env/fullchain.pem" || keyPath != "/env/key.pem" {
		t.Fatalf("resolveCertificatePaths() = %q, %q, want acme.sh environment values", certPath, keyPath)
	}

	certPath, keyPath, err = resolveCertificatePaths(CallerAcmeSH, "/flag/cert.pem", "/flag/key.pem")
	if err != nil {
		t.Fatalf("resolveCertificatePaths() with flags error = %v", err)
	}
	if certPath != "/flag/cert.pem" || keyPath != "/flag/key.pem" {
		t.Fatalf("resolveCertificatePaths() = %q, %q, want flag values", certPath, keyPath)
	}

	if _, _, err := resolveCertificatePaths(CallerCLI, "", "/key.pem"); err == nil {
		t.Fatal("resolveCertificatePaths(cli) error = nil, want missing cert error")
	}
}

func TestResolveCertificatePathsLego(t *testing.T) {
	t.Setenv("LEGO_HOOK_CERT_PATH", "/lego/cert.pem")
	t.Setenv("LEGO_HOOK_CERT_KEY_PATH", "/lego/key.pem")

	certPath, keyPath, err := resolveCertificatePaths(CallerLego, "", "")
	if err != nil {
		t.Fatalf("resolveCertificatePaths() error = %v", err)
	}
	if certPath != "/lego/cert.pem" || keyPath != "/lego/key.pem" {
		t.Fatalf("resolveCertificatePaths() = %q, %q, want lego hook environment values", certPath, keyPath)
	}
}

func TestLoadCertData(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "fullchain.pem")
	keyPath := filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certPath, []byte("certificate data"), 0o600); err != nil {
		t.Fatalf("WriteFile(cert) error = %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("private key data"), 0o600); err != nil {
		t.Fatalf("WriteFile(key) error = %v", err)
	}

	data, err := loadCertData(certificateInput{
		Domain:   "example.com",
		CertPath: certPath,
		KeyPath:  keyPath,
	})
	if err != nil {
		t.Fatalf("loadCertData() error = %v", err)
	}
	if data.Domain != "example.com" || data.FullChain != "certificate data" || data.PrivateKey != "private key data" {
		t.Fatalf("loadCertData() = %#v, want populated certificate data", data)
	}
}
