package cryptoutil

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptLargeMessage(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	message := bytes.Repeat([]byte("metrics payload "), 200)
	encrypted, err := Encrypt(&privateKey.PublicKey, message)
	require.NoError(t, err)
	require.NotEqual(t, message, encrypted)
	require.Greater(t, len(encrypted), privateKey.Size())

	decrypted, err := Decrypt(privateKey, encrypted)
	require.NoError(t, err)
	require.Equal(t, message, decrypted)
}

func TestLoadRSAKeys(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	directory := t.TempDir()
	publicKeyPath := filepath.Join(directory, "public.pem")
	privateKeyPath := filepath.Join(directory, "private.pem")

	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(publicKeyPath, pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicDER,
	}), 0o600))
	require.NoError(t, os.WriteFile(privateKeyPath, pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}), 0o600))

	loadedPublicKey, err := LoadPublicKey(publicKeyPath)
	require.NoError(t, err)
	require.Equal(t, privateKey.PublicKey.N, loadedPublicKey.N)
	require.Equal(t, privateKey.PublicKey.E, loadedPublicKey.E)

	loadedPrivateKey, err := LoadPrivateKey(privateKeyPath)
	require.NoError(t, err)
	require.Equal(t, privateKey.D, loadedPrivateKey.D)
}
