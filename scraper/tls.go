// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/Liamraystanley/marill

package scraper

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"net"
	"strings"
	"time"
)

// TLSResponse is the TLS/SSL handshake response and certificate information.
type TLSResponse struct {
	HandshakeComplete bool
	PeerCertificates  []*ResponseCert
	VerifiedChains    [][]*ResponseCert
}

type ResponseCert struct {
	Version        int
	NotBefore      time.Time
	NotAfter       time.Time
	Issuer         *CertName
	Subject        *CertName
	DNSNames       []string
	EmailAddresses []string
	IPAddresses    []net.IP
}

type CertName struct {
	Country       string
	Organization  string
	Locality      string
	Province      string
	StreetAddress string
	CommonName    string
}

func tlsToShort(data *tls.ConnectionState) *TLSResponse {
	if data == nil {
		return nil
	}

	ssl := &TLSResponse{
		HandshakeComplete: data.HandshakeComplete,
		PeerCertificates:  make([]*ResponseCert, len(data.PeerCertificates)),
		VerifiedChains:    make([][]*ResponseCert, len(data.VerifiedChains)),
	}

	// loop through the peer certs first
	for i := 0; i < len(data.PeerCertificates); i++ {
		ssl.PeerCertificates[i] = tlsCertToShort(data.PeerCertificates[i])
	}

	// and now verified certs
	for i := 0; i < len(data.VerifiedChains); i++ {
		ssl.VerifiedChains[i] = make([]*ResponseCert, len(data.VerifiedChains[i]))

		for c := 0; c < len(data.VerifiedChains[i]); c++ {
			ssl.VerifiedChains[i][c] = tlsCertToShort(data.VerifiedChains[i][c])
		}
	}

	return ssl
}

func tlsCertToShort(data *x509.Certificate) *ResponseCert {
	if data == nil {
		return nil
	}

	ssl := &ResponseCert{
		Version:        data.Version,
		Issuer:         tlsNameToShort(data.Issuer),
		Subject:        tlsNameToShort(data.Subject),
		NotBefore:      data.NotBefore,
		NotAfter:       data.NotAfter,
		DNSNames:       data.DNSNames,
		EmailAddresses: data.EmailAddresses,
		IPAddresses:    data.IPAddresses,
	}

	return ssl
}

func tlsNameToShort(data pkix.Name) *CertName {
	name := &CertName{CommonName: data.CommonName}

	if data.Country != nil {
		name.Country = strings.Join(data.Country, ", ")
	}

	if data.Organization != nil {
		name.Organization = strings.Join(data.Organization, ", ")
	}

	if data.Locality != nil {
		name.Locality = strings.Join(data.Locality, ", ")
	}

	if data.Province != nil {
		name.Province = strings.Join(data.Province, ", ")
	}

	if data.StreetAddress != nil {
		name.StreetAddress = strings.Join(data.StreetAddress, ", ")
	}

	return name
}
