package cryptoutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("decode public key: PEM block not found")
	}

	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		publicKey, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("decode public key: key is not RSA")
		}
		return publicKey, nil
	}

	if publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return publicKey, nil
	}

	return nil, fmt.Errorf("decode public key: unsupported RSA key format")
}

func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("decode private key: PEM block not found")
	}

	if privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return privateKey, nil
	}

	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		privateKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("decode private key: key is not RSA")
		}
		return privateKey, nil
	}

	return nil, fmt.Errorf("decode private key: unsupported RSA key format")
}

func Encrypt(publicKey *rsa.PublicKey, data []byte) ([]byte, error) {
	if publicKey == nil {
		return nil, fmt.Errorf("encrypt: public key is nil")
	}
	if len(data) == 0 {
		return []byte{}, nil
	}

	chunkSize := publicKey.Size() - 2*sha256.Size - 2
	if chunkSize <= 0 {
		return nil, fmt.Errorf("encrypt: RSA key is too small")
	}

	encrypted := make([]byte, 0, ((len(data)+chunkSize-1)/chunkSize)*publicKey.Size())
	for len(data) > 0 {
		partSize := min(chunkSize, len(data))
		block, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, data[:partSize], nil)
		if err != nil {
			return nil, fmt.Errorf("encrypt RSA block: %w", err)
		}
		encrypted = append(encrypted, block...)
		data = data[partSize:]
	}

	return encrypted, nil
}

func Decrypt(privateKey *rsa.PrivateKey, data []byte) ([]byte, error) {
	if privateKey == nil {
		return nil, fmt.Errorf("decrypt: private key is nil")
	}
	if len(data) == 0 {
		return []byte{}, nil
	}

	blockSize := privateKey.Size()
	if len(data)%blockSize != 0 {
		return nil, fmt.Errorf("decrypt: ciphertext length %d is not a multiple of RSA block size %d", len(data), blockSize)
	}

	decrypted := make([]byte, 0, len(data))
	for len(data) > 0 {
		block, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, data[:blockSize], nil)
		if err != nil {
			return nil, fmt.Errorf("decrypt RSA block: %w", err)
		}
		decrypted = append(decrypted, block...)
		data = data[blockSize:]
	}

	return decrypted, nil
}
