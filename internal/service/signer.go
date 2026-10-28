package service

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"go.mozilla.org/pkcs7"
)

type ProfileSigner interface {
	Sign(rawXML []byte) ([]byte, error)
}

type PKCS7Signer struct {
	leaf          *x509.Certificate
	privateKey    crypto.PrivateKey
	intermediates []*x509.Certificate
}

// NewPKCS7SignerFromFiles loads an SSL certificate chain and private key from file paths.
func NewPKCS7SignerFromFiles(certPath, keyPath string) (*PKCS7Signer, error) {
	if certPath == "" || keyPath == "" {
		return nil, nil
	}

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read signing cert from %s: %w", certPath, err)
	}

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read signing key from %s: %w", keyPath, err)
	}

	return NewPKCS7Signer(certPEM, keyPEM)
}

// NewPKCS7Signer parses PEM-encoded certificate chain and private key.
func NewPKCS7Signer(certPEM, keyPEM []byte) (*PKCS7Signer, error) {
	var certs []*x509.Certificate
	rest := certPEM
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			c, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("invalid certificate in chain: %w", err)
			}
			certs = append(certs, c)
		}
	}

	if len(certs) == 0 {
		return nil, errors.New("no certificates found in cert pem data")
	}

	leaf := certs[0]
	var intermediates []*x509.Certificate
	if len(certs) > 1 {
		intermediates = certs[1:]
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, errors.New("no private key found in key pem data")
	}

	var privKey crypto.PrivateKey
	if k, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes); err == nil {
		privKey = k
	} else if k, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes); err == nil {
		privKey = k
	} else if k, err := x509.ParseECPrivateKey(keyBlock.Bytes); err == nil {
		privKey = k
	} else {
		return nil, errors.New("unsupported private key format; expected RSA or ECDSA in PKCS#1, PKCS#8 or SEC1 format")
	}

	return &PKCS7Signer{
		leaf:          leaf,
		privateKey:    privKey,
		intermediates: intermediates,
	}, nil
}

// Sign wraps raw XML content in a PKCS#7 / CMS non-detached signed structure.
func (s *PKCS7Signer) Sign(rawXML []byte) ([]byte, error) {
	if s == nil || s.leaf == nil {
		return rawXML, nil
	}

	signedData, err := pkcs7.NewSignedData(rawXML)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize pkcs7 signed data: %w", err)
	}

	signedData.SetDigestAlgorithm(pkcs7.OIDDigestAlgorithmSHA256)

	if err := signedData.AddSigner(s.leaf, s.privateKey, pkcs7.SignerInfoConfig{}); err != nil {
		return nil, fmt.Errorf("failed to add signer: %w", err)
	}

	for _, inter := range s.intermediates {
		signedData.AddCertificate(inter)
	}

	return signedData.Finish()
}
