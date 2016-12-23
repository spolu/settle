package cert

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"strings"
	"time"

	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/logging"
)

// GetGetCertificate computes the GetCertificate function to serve TLS securily
// in production using LetsEncrypt and insecurely in QA using a self signed
// certificate.
func GetGetCertificate(
	ctx context.Context,
	host string,
	certFile string,
	keyFile string,
) func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	var cert *tls.Certificate
	var err error

	switch env.Get(ctx).Environment {
	case env.Production:
		// In Production use specified certificates
		cert, err = GetCertificateFromFiles(ctx, certFile, keyFile)
	case env.QA:
		cert, err = GetSelfSignedQACertificate(ctx, host)
	}

	return func(
		*tls.ClientHelloInfo,
	) (*tls.Certificate, error) {
		if err != nil {
			return nil, err
		}
		return cert, nil
	}
}

// GetCertificateFromFiles retrieves the certificate from the specified files.
func GetCertificateFromFiles(
	ctx context.Context,
	certFile string,
	keyFile string,
) (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	logging.Logf(ctx,
		"Loading certificate: crt_file=%s, key_file=%s", certFile, keyFile)

	return &cert, nil
}

// GetSelfSignedQACertificate returns a self signed certificate for the host
// passed in QA. QA client do not verify certificates.
func GetSelfSignedQACertificate(
	ctx context.Context,
	host string,
) (*tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"QA Mint (invalid)"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		IsCA: true,
		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature |
			x509.KeyUsageCertSign,

		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	h := strings.Split(host, ":")[0]
	if ip := net.ParseIP(h); ip != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
		logging.Logf(ctx, "Self-signing QA certificate: ip=%s", ip)

	} else {
		template.DNSNames = append(template.DNSNames, h)
		logging.Logf(ctx, "Self-signing QA certificate: dns=%s", h)
	}

	bytes, err := x509.CreateCertificate(
		rand.Reader, &template, &template, priv.Public(), priv)
	if err != nil {
		return nil, err
	}

	return &tls.Certificate{
		Certificate: [][]byte{bytes},
		PrivateKey:  priv,
	}, nil
}
