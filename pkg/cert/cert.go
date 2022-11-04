package cert

import (
	"bytes"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

// Config from which certificates are generated
type CertConfig struct {
  // Set these
  Name      string
  Namespace string
  Org       string
  // Internal variables
  caCert    *x509.Certificate
  caKey     *rsa.PrivateKey
}

// Generate CA cert PEM file
func (c *CertConfig) GenerateCACert() (*bytes.Buffer, error){
  // We only need one CA
  if c.caCert != nil {
    return nil, fmt.Errorf("the CA is already created")
  }

	c.caCert = &x509.Certificate{
		SerialNumber: big.NewInt(2020),
		Subject: pkix.Name{
			Organization: []string{c.Org},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	// CA private key
  var err error
	c.caKey, err = rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		fmt.Println(err)
	}

	// Self signed CA certificate
	caBytes, err := x509.CreateCertificate(cryptorand.Reader, c.caCert, c.caCert, &c.caKey.PublicKey, c.caKey)
	if err != nil {
		fmt.Println(err)
	}

	// PEM encode CA cert
  caPEM := new(bytes.Buffer)
	_ = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
  return caPEM, nil
}

// Generate a Certificate and PrivateKey PEM file
func (c *CertConfig) GenerateServerCert() (*bytes.Buffer, *bytes.Buffer, error){
  // No server cert without a CA
  if c.caCert == nil {
    return nil, nil, fmt.Errorf("cant generate a server certificate before ca")
  }
  // CommonName for certificate
	commonName := fmt.Sprintf("%s.%s.svc", c.Name, c.Namespace)
  // Generate DNS entries
	dnsNames := []string{
    c.Name,
		fmt.Sprintf("%s.%s", c.Name, c.Namespace),
    commonName,
  }
  cert := &x509.Certificate{
		DNSNames:     dnsNames,
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{c.Org},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	// server private key
	serverPrivKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		fmt.Println(err)
	}

	// sign the server cert
	serverCertBytes, err := x509.CreateCertificate(cryptorand.Reader, cert, c.caCert, &serverPrivKey.PublicKey, c.caKey)
	if err != nil {
		fmt.Println(err)
	}

	// PEM encode the  server cert and key
  serverCertPEM := new(bytes.Buffer)
	_ = pem.Encode(serverCertPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: serverCertBytes,
	})

  serverPrivKeyPEM := new(bytes.Buffer)
	_ = pem.Encode(serverPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(serverPrivKey),
	})

  return serverCertPEM, serverPrivKeyPEM, nil
}
