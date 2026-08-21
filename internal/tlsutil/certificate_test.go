package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSelfSigned(t *testing.T) {
	directory := t.TempDir()
	certFile := filepath.Join(directory, "server.crt")
	keyFile := filepath.Join(directory, "server.key")

	err := GenerateSelfSigned(CertificateOptions{
		CertFile: certFile,
		KeyFile:  keyFile,
		Hosts:    []string{"localhost", "127.0.0.1", "::1"},
		ValidFor: 24 * time.Hour,
	})
	require.NoError(t, err)

	_, err = tls.LoadX509KeyPair(certFile, keyFile)
	require.NoError(t, err)

	certificatePEM, err := os.ReadFile(certFile)
	require.NoError(t, err)
	block, _ := pem.Decode(certificatePEM)
	require.NotNil(t, block)
	certificate, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	assert.NoError(t, certificate.VerifyHostname("localhost"))
	assert.NoError(t, certificate.VerifyHostname("127.0.0.1"))
	assert.NoError(t, certificate.VerifyHostname("::1"))
	assert.Contains(t, certificate.ExtKeyUsage, x509.ExtKeyUsageServerAuth)

	keyInfo, err := os.Stat(keyFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), keyInfo.Mode().Perm())

	_, err = LoadServerCredentials(certFile, keyFile)
	assert.NoError(t, err)
	_, err = LoadClientCredentials(certFile, "localhost")
	assert.NoError(t, err)
}

func TestGenerateSelfSignedValidatesOptions(t *testing.T) {
	tests := []struct {
		name    string
		options CertificateOptions
	}{
		{name: "empty certificate path", options: CertificateOptions{KeyFile: "key", Hosts: []string{"localhost"}, ValidFor: time.Hour}},
		{name: "empty key path", options: CertificateOptions{CertFile: "cert", Hosts: []string{"localhost"}, ValidFor: time.Hour}},
		{name: "empty hosts", options: CertificateOptions{CertFile: "cert", KeyFile: "key", ValidFor: time.Hour}},
		{name: "invalid validity", options: CertificateOptions{CertFile: "cert", KeyFile: "key", Hosts: []string{"localhost"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, GenerateSelfSigned(tt.options))
		})
	}
}

func TestLoadClientCredentialsRejectsInvalidPEM(t *testing.T) {
	certFile := filepath.Join(t.TempDir(), "invalid.crt")
	require.NoError(t, os.WriteFile(certFile, []byte("not a certificate"), 0o600))

	_, err := LoadClientCredentials(certFile, "localhost")
	assert.Error(t, err)
}
