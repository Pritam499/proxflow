package pinning

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/Pritam499/proxflow/v2"
)

func TestCertPinning_Provision(t *testing.T) {
	cp := &CertPinning{
		Pins: []string{"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"},
	}
	ctx := caddy.ActiveContext()
	err := cp.Provision(ctx)
	if err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	if cp.PinType != "public_key" {
		t.Errorf("Expected PinType = public_key, got %s", cp.PinType)
	}
	if cp.Action != "reject" {
		t.Errorf("Expected Action = reject, got %s", cp.Action)
	}
}

func TestCertPinning_ValidatePeerCertificate(t *testing.T) {
	// Generate a test certificate
	cert, pin := generateTestCert(t)

	cp := &CertPinning{
		PinType: "public_key",
		Pins:    []string{pin},
	}
	cp.Provision(caddy.ActiveContext())

	// This should pass
	err := cp.ValidatePeerCertificate([][]byte{cert.Raw}, nil)
	if err != nil {
		t.Errorf("Expected validation to pass, got error: %v", err)
	}

	// Test with wrong pin
	cp.Pins = []string{"wrongpin1234567890abcdef1234567890abcdef1234567890abcdef1234567890"}
	err = cp.ValidatePeerCertificate([][]byte{cert.Raw}, nil)
	if err == nil {
		t.Error("Expected validation to fail with wrong pin")
	}

	// Test log action
	cp.Action = "log"
	cp.Pins = []string{"wrongpin1234567890abcdef1234567890abcdef1234567890abcdef1234567890"}
	err = cp.ValidatePeerCertificate([][]byte{cert.Raw}, nil)
	// Should not error in log mode
	if err != nil {
		t.Errorf("Expected no error in log mode, got: %v", err)
	}
}

func generateTestCert(t *testing.T) (*x509.Certificate, string) {
	// Generate private key
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	// Calculate pin
	cp := &CertPinning{PinType: "public_key"}
	pin, err := cp.pinPublicKey(certDER)
	if err != nil {
		t.Fatalf("Failed to calculate pin: %v", err)
	}

	return cert, pin
}