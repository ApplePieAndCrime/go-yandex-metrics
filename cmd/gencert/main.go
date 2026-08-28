// Команда gencert создаёт самоподписанный сертификат для gRPC-сервера.
package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/tlsutil"
)

func main() {
	certFile := flag.String("cert", "server.crt", "путь для сохранения сертификата")
	keyFile := flag.String("key", "server.key", "путь для сохранения приватного ключа")
	hosts := flag.String("hosts", "localhost,127.0.0.1,::1", "имена хостов и IP-адреса через запятую")
	validFor := flag.Duration("valid-for", 365*24*time.Hour, "срок действия сертификата")
	flag.Parse()

	if err := tlsutil.GenerateSelfSigned(tlsutil.CertificateOptions{
		CertFile: *certFile,
		KeyFile:  *keyFile,
		Hosts:    strings.Split(*hosts, ","),
		ValidFor: *validFor,
	}); err != nil {
		log.Fatal(err)
	}

	log.Printf("Сертификат сохранён в %s, приватный ключ — в %s", *certFile, *keyFile)
}
