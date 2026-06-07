package httpclient

import (
	"crypto/tls"
	"net/http"
)

func NewClient(tlsMin string) *http.Client {
	var minTLSVersion uint16
	switch tlsMin {
	case "1.1":
		minTLSVersion = tls.VersionTLS11
	case "1.2":
		minTLSVersion = tls.VersionTLS12
	default:
		minTLSVersion = tls.VersionTLS13
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: minTLSVersion,
			},
		},
	}
}
