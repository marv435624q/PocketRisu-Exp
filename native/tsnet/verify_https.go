package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/url"
)

type httpsVerification struct {
	Type        string `json:"type"`
	Success     bool   `json:"success"`
	TLSVerified bool   `json:"tlsVerified"`
	HTTPStatus  int    `json:"httpStatus"`
	ErrorCode   string `json:"errorCode,omitempty"`
	NetworkClass string `json:"networkClass,omitempty"`
}

// Probe the already listening server through tsnet's dialer. Production roots
// are nil (system trust); tests may inject a synthetic CA. No redirect, auth
// header, cookie, inherited proxy, response body or actual hostname is logged.
func verifyHTTPS(ctx context.Context, domain string,
	dial func(context.Context, string, string) (net.Conn, error), roots *x509.CertPool) httpsVerification {
	result := httpsVerification{Type: "https-verification"}
	transport := &http.Transport{DialContext: dial,
		TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	u := url.URL{Scheme: "https", Host: domain, Path: "/"}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		result.ErrorCode = "request"
		return result
	}
	res, err := client.Do(req)
	if err != nil {
		result.NetworkClass = networkErrorClass(err)
		var unknownAuthority x509.UnknownAuthorityError
		var hostnameError x509.HostnameError
		var invalidCertificate x509.CertificateInvalidError
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			result.ErrorCode = "deadline"
		case errors.As(err, &unknownAuthority), errors.As(err, &hostnameError), errors.As(err, &invalidCertificate):
			result.ErrorCode = "certificate"
		default:
			result.ErrorCode = "https-request"
		}
		return result
	}
	defer res.Body.Close()
	result.HTTPStatus = res.StatusCode
	result.TLSVerified = res.TLS != nil && len(res.TLS.VerifiedChains) != 0
	result.Success = result.TLSVerified && res.StatusCode >= 200 && res.StatusCode < 500
	if !result.Success {
		result.ErrorCode = "http-response"
	}
	return result
}
