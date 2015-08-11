package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"time"
)

var (
	flagListen      = flag.String("listen", ":4443", "socket address to listen to")
	flagBackend     = flag.String("backend", "localhost:8080", "backend server to proxy")
	flagHost        = flag.String("host", "localhost", "hostname for certificate")
	flagSSLCertPath = flag.String("cert", "", "path to SSL PEM certificate file; defaults to a generated self-signed cert")
	flagSSLKeyPath  = flag.String("key", "", "path to the SSL Key file PEM for the cert")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "simplehttpsproxy [flags]\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Host = *flagBackend
			req.URL.Scheme = "http"
			req.Header.Set("X-Forwarded-Proto", "https")
		},
	}

	var cert *tls.Certificate
	var err error
	if *flagSSLCertPath == "" || *flagSSLKeyPath == "" {
		if *flagSSLCertPath != "" || *flagSSLKeyPath != "" {
			log.Fatalf("cannot specify -cert without -key")
		}
		if cert, err = genSelfSignedCert(*flagHost); err != nil {
			log.Fatalf("cannot generate cert: %s", err)
		}
		log.Printf("starting proxying %s on %s with generated cert", *flagBackend, *flagListen)
		panic(listenAndServeTLS(*flagListen, cert, proxy))
	}

	log.Printf("starting proxying %s on %s with given cert", *flagBackend, *flagListen)
	panic(http.ListenAndServeTLS(*flagListen, *flagSSLCertPath, *flagSSLKeyPath, proxy))
}

func genSelfSignedCert(host string) (*tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %s", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Fictitious Co"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
	} else {
		template.DNSNames = append(template.DNSNames, host)
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate self-signed certificate: %s", err)
	}
	return &tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}, nil
}

func listenAndServeTLS(addr string, cert *tls.Certificate, handler http.Handler) error {
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	tlsConfig := &tls.Config{
		NextProtos:   []string{"http/1.1"},
		Certificates: []tls.Certificate{*cert},
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	// This is less robust than stdlib, but this is ok for this kind of program
	// (stdlib uses TCP keep-alive timeouts to clean up dead conns)
	tlsListener := tls.NewListener(ln.(*net.TCPListener), tlsConfig)
	return server.Serve(tlsListener)
}
