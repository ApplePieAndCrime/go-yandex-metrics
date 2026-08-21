// Пакет tlsutil содержит вспомогательные функции для настройки TLS.
package tlsutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc/credentials"
)

// CertificateOptions задаёт параметры самоподписанного сертификата.
type CertificateOptions struct {
	CertFile string
	KeyFile  string
	Hosts    []string
	ValidFor time.Duration
}

// GenerateSelfSigned создаёт самоподписанный TLS-сертификат и приватный ключ.
func GenerateSelfSigned(options CertificateOptions) error {
	if options.CertFile == "" {
		return fmt.Errorf("путь к сертификату не задан")
	}
	if options.KeyFile == "" {
		return fmt.Errorf("путь к приватному ключу не задан")
	}
	if options.ValidFor <= 0 {
		return fmt.Errorf("срок действия сертификата должен быть положительным")
	}

	dnsNames, ipAddresses := splitHosts(options.Hosts)
	if len(dnsNames) == 0 && len(ipAddresses) == 0 {
		return fmt.Errorf("необходимо указать хотя бы один хост или IP-адрес")
	}
	commonName := ""
	if len(dnsNames) > 0 {
		commonName = dnsNames[0]
	} else {
		commonName = ipAddresses[0].String()
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("создать приватный ключ: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return fmt.Errorf("создать серийный номер сертификата: %w", err)
	}

	now := time.Now()
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"go-yandex-metrics"},
			CommonName:   commonName,
		},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(options.ValidFor),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
	}

	certificateDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("создать сертификат: %w", err)
	}

	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("закодировать приватный ключ: %w", err)
	}

	certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateDER})
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyDER})

	if err := os.WriteFile(options.CertFile, certificatePEM, 0o644); err != nil {
		return fmt.Errorf("записать сертификат: %w", err)
	}
	if err := os.WriteFile(options.KeyFile, privateKeyPEM, 0o600); err != nil {
		return fmt.Errorf("записать приватный ключ: %w", err)
	}
	if err := os.Chmod(options.KeyFile, 0o600); err != nil {
		return fmt.Errorf("установить права приватного ключа: %w", err)
	}

	return nil
}

// LoadServerCredentials загружает сертификат и ключ gRPC-сервера.
func LoadServerCredentials(certFile, keyFile string) (credentials.TransportCredentials, error) {
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("загрузить TLS-сертификат сервера: %w", err)
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{certificate},
		MinVersion:   tls.VersionTLS13,
	}), nil
}

// LoadClientCredentials создаёт TLS-настройки клиента с доверенным сертификатом сервера.
func LoadClientCredentials(certFile, serverName string) (credentials.TransportCredentials, error) {
	certificatePEM, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("прочитать TLS-сертификат сервера: %w", err)
	}

	rootCAs := x509.NewCertPool()
	if !rootCAs.AppendCertsFromPEM(certificatePEM) {
		return nil, fmt.Errorf("добавить TLS-сертификат сервера: PEM-сертификат не найден")
	}

	return credentials.NewTLS(&tls.Config{
		RootCAs:    rootCAs,
		ServerName: serverName,
		MinVersion: tls.VersionTLS13,
	}), nil
}

func splitHosts(hosts []string) ([]string, []net.IP) {
	dnsNames := make([]string, 0, len(hosts))
	ipAddresses := make([]net.IP, 0, len(hosts))

	for _, host := range hosts {
		host = strings.TrimSpace(host)
		if host == "" {
			continue
		}
		if ip := net.ParseIP(host); ip != nil {
			ipAddresses = append(ipAddresses, ip)
			continue
		}
		dnsNames = append(dnsNames, host)
	}

	return dnsNames, ipAddresses
}
