// Copyright 2023 The ProxFlow Authors
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

package pinning

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"fmt"

	"github.com/Pritam499/proxflow/v2"
)

func init() {
	caddy.RegisterModule(CertPinning{})
}

// CertPinning implements certificate pinning for TLS connections.
type CertPinning struct {
	// PinType specifies the type of pinning: "public_key" or "certificate".
	PinType string `json:"pin_type,omitempty"`

	// Pins is a list of expected pin values (hex-encoded SHA256 hashes).
	Pins []string `json:"pins,omitempty"`

	// Action specifies what to do on pin mismatch: "reject" or "log".
	Action string `json:"action,omitempty"`
}

// CaddyModule returns the Caddy module information.
func (CertPinning) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "tls.certificates.pinning",
		New: func() caddy.Module { return new(CertPinning) },
	}
}

// Provision sets up the certificate pinning.
func (cp *CertPinning) Provision(ctx caddy.Context) error {
	if cp.PinType == "" {
		cp.PinType = "public_key"
	}
	if cp.Action == "" {
		cp.Action = "reject"
	}

	// Validate pin values
	for _, pin := range cp.Pins {
		if len(pin) != 64 { // SHA256 hex is 64 characters
			return fmt.Errorf("invalid pin length: %s (expected 64 hex characters)", pin)
		}
		if _, err := hex.DecodeString(pin); err != nil {
			return fmt.Errorf("invalid pin format: %s (expected hex)", pin)
		}
	}

	return nil
}

// ValidatePeerCertificate validates the peer certificate against pinned values.
func (cp *CertPinning) ValidatePeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	if len(rawCerts) == 0 {
		return fmt.Errorf("no certificates provided")
	}

	var pinValue string
	var err error

	switch cp.PinType {
	case "public_key":
		pinValue, err = cp.pinPublicKey(rawCerts[0])
	case "certificate":
		pinValue, err = cp.pinCertificate(rawCerts[0])
	default:
		return fmt.Errorf("unsupported pin type: %s", cp.PinType)
	}

	if err != nil {
		return err
	}

	// Check if pin matches any expected pin
	for _, expectedPin := range cp.Pins {
		if pinValue == expectedPin {
			return nil // Pin matches
		}
	}

	// Pin mismatch
	err = fmt.Errorf("certificate pin mismatch: got %s", pinValue)

	if cp.Action == "log" {
		// In a real implementation, log the error
		// For now, just return nil to allow the connection
		return nil
	}

	return err
}

// pinPublicKey calculates the pin for the certificate's public key.
func (cp *CertPinning) pinPublicKey(rawCert []byte) (string, error) {
	cert, err := x509.ParseCertificate(rawCert)
	if err != nil {
		return "", err
	}

	// Get the public key in DER format
	publicKeyDER, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return "", err
	}

	// Calculate SHA256 hash
	hash := sha256.Sum256(publicKeyDER)
	return hex.EncodeToString(hash[:]), nil
}

// pinCertificate calculates the pin for the entire certificate.
func (cp *CertPinning) pinCertificate(rawCert []byte) (string, error) {
	// Calculate SHA256 hash of the certificate
	hash := sha256.Sum256(rawCert)
	return hex.EncodeToString(hash[:]), nil
}

// Interface guards
var (
	_ caddy.Module = (*CertPinning)(nil)
)