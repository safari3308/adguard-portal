package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.mozilla.org/pkcs7"
)

func generateTestCertAndKey(t *testing.T, isRSA bool) ([]byte, []byte) {
	t.Helper()
	var privKey any
	var pubKey any
	var err error

	if isRSA {
		rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("failed to generate rsa key: %v", err)
		}
		privKey = rsaKey
		pubKey = &rsaKey.PublicKey
	} else {
		ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatalf("failed to generate ec key: %v", err)
		}
		privKey = ecKey
		pubKey = &ecKey.PublicKey
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test.example.com",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, pubKey, privKey)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	var keyPEM []byte
	if isRSA {
		keyPEM = pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privKey.(*rsa.PrivateKey)),
		})
	} else {
		ecBytes, err := x509.MarshalECPrivateKey(privKey.(*ecdsa.PrivateKey))
		if err != nil {
			t.Fatalf("failed to marshal ec key: %v", err)
		}
		keyPEM = pem.EncodeToMemory(&pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: ecBytes,
		})
	}

	return certPEM, keyPEM
}

func TestPKCS7Signer(t *testing.T) {
	certPEM, keyPEM := generateTestCertAndKey(t, true)
	signer, err := NewPKCS7Signer(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("NewPKCS7Signer failed: %v", err)
	}

	xmlData := []byte("<plist><dict><key>test</key><string>value</string></dict></plist>")
	signed, err := signer.Sign(xmlData)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if len(signed) <= len(xmlData) {
		t.Fatalf("expected signed data to be larger than raw xml")
	}

	// Verify using pkcs7.Parse
	p7, err := pkcs7.Parse(signed)
	if err != nil {
		t.Fatalf("failed to parse generated pkcs7: %v", err)
	}
	if string(p7.Content) != string(xmlData) {
		t.Errorf("encapsulated content mismatch, expected %s, got %s", string(xmlData), string(p7.Content))
	}

	// Test ECDSA key
	ecCertPEM, ecKeyPEM := generateTestCertAndKey(t, false)
	ecSigner, err := NewPKCS7Signer(ecCertPEM, ecKeyPEM)
	if err != nil {
		t.Fatalf("NewPKCS7Signer with ECDSA failed: %v", err)
	}
	signedEC, err := ecSigner.Sign(xmlData)
	if err != nil {
		t.Fatalf("Sign with ECDSA failed: %v", err)
	}
	if len(signedEC) == 0 {
		t.Errorf("expected signed data from ECDSA signer")
	}

	// Test NewPKCS7SignerFromFiles
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")
	if err := os.WriteFile(certFile, certPEM, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		t.Fatal(err)
	}

	fileSigner, err := NewPKCS7SignerFromFiles(certFile, keyFile)
	if err != nil || fileSigner == nil {
		t.Fatalf("NewPKCS7SignerFromFiles failed: %v", err)
	}

	// Test empty file paths
	emptySigner, err := NewPKCS7SignerFromFiles("", "")
	if err != nil || emptySigner != nil {
		t.Errorf("expected nil signer and nil err for empty paths, got: %v, %v", emptySigner, err)
	}

	// Test non-existent file
	_, err = NewPKCS7SignerFromFiles("/nonexistent/cert.pem", keyFile)
	if err == nil {
		t.Errorf("expected error for non-existent cert file, got nil")
	}
	_, err = NewPKCS7SignerFromFiles(certFile, "/nonexistent/key.pem")
	if err == nil {
		t.Errorf("expected error for non-existent key file, got nil")
	}

	// Test invalid PEM
	_, err = NewPKCS7Signer([]byte("invalid cert"), keyPEM)
	if err == nil {
		t.Errorf("expected error for invalid cert PEM, got nil")
	}
	_, err = NewPKCS7Signer(certPEM, []byte("invalid key"))
	if err == nil {
		t.Errorf("expected error for invalid key PEM, got nil")
	}

	// Test nil signer Sign() fallback
	var nilSigner *PKCS7Signer
	rawOut, err := nilSigner.Sign(xmlData)
	if err != nil || string(rawOut) != string(xmlData) {
		t.Errorf("expected nil signer to return raw data unmodified")
	}
}
