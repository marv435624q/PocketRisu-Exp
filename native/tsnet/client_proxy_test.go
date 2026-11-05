package main

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testProxyPassword = "synthetic-test-password-32-characters"
const testProxyRealm = "PocketRisu-synthetic-test-realm"

func testAuth() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte("pocketrisu:"+testProxyPassword))
}

func TestClientProxyStreamingBeforeCompletion(t *testing.T) {
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: first\n\n")
		w.(http.Flusher).Flush()
		select {
		case <-release:
		case <-r.Context().Done():
		}
		io.WriteString(w, "data: last\n\n")
	}))
	defer upstream.Close()
	defer close(release)
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, strings.TrimPrefix(upstream.URL, "http://"))
	}
	p, _ := newClientProxy("phone.example.ts.net:80", testProxyPassword, testProxyRealm, dial, dial)
	defer p.Close()
	s := httptest.NewServer(p)
	defer s.Close()
	u, _ := url.Parse(s.URL)
	transport := &http.Transport{Proxy: http.ProxyURL(u)}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	r, _ := http.NewRequest("GET", "http://phone.example.ts.net/stream", nil)
	r.Header.Set("Proxy-Authorization", testAuth())
	resp, err := client.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	line, err := bufio.NewReader(resp.Body).ReadString('\n')
	if err != nil || line != "data: first\n" {
		t.Fatalf("stream buffered until completion: %q %v", line, err)
	}
}

// Explicit opt-in local browser fixture. Neither Tailscale nor a real account is used.
func TestClientProxyBrowserFixture(t *testing.T) {
	dir := os.Getenv("POCKETRISU_BROWSER_FIXTURE")
	if dir == "" {
		t.Skip("local browser fixture not requested")
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Proxy-Authorization") != "" {
			http.Error(w, "credential leak", 500)
			return
		}
		if r.URL.Path == "/post" {
			body, _ := io.ReadAll(r.Body)
			fmt.Fprintf(w, "%s|%s|%s|%s", r.Method, r.URL.RawQuery, r.Header.Get("Cookie"), body)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, "<!doctype html><title>Synthetic Tailnet proxy</title><p>Local fixture</p>")
	}))
	defer upstream.Close()
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, strings.TrimPrefix(upstream.URL, "http://"))
	}
	p, _ := newClientProxy("phone.example.ts.net:80", testProxyPassword, testProxyRealm, dial, dial)
	defer p.Close()
	s := httptest.NewServer(p)
	defer s.Close()
	if err := os.WriteFile(filepath.Join(dir, "browser-proxy-port.txt"), []byte(strings.TrimPrefix(s.URL, "http://127.0.0.1:")), 0600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(dir, "browser-fixture-stop")); err == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("browser fixture timed out")
}

func TestClientProxyWebsocketUpgradeCloses(t *testing.T) {
	closed := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, b, _ := w.(http.Hijacker).Hijack()
		defer c.Close()
		defer close(closed)
		b.WriteString("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n")
		b.Flush()
		io.Copy(c, c)
	}))
	defer upstream.Close()
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, strings.TrimPrefix(upstream.URL, "http://"))
	}
	p, _ := newClientProxy("phone.example.ts.net:80", testProxyPassword, testProxyRealm, dial, dial)
	defer p.Close()
	s := httptest.NewServer(p)
	defer s.Close()
	c, err := net.Dial("tcp", strings.TrimPrefix(s.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(5 * time.Second))
	fmt.Fprintf(c, "GET http://phone.example.ts.net/ws HTTP/1.1\r\nHost: phone.example.ts.net\r\nConnection: Upgrade\r\nUpgrade: websocket\r\nProxy-Authorization: %s\r\n\r\n", testAuth())
	reader := bufio.NewReader(c)
	response, err := http.ReadResponse(reader, &http.Request{Method: "GET"})
	if err != nil || response.StatusCode != 101 {
		t.Fatalf("upgrade failed: %v %v", response, err)
	}
	c.Write([]byte("websocket-frame"))
	got := make([]byte, len("websocket-frame"))
	if _, err = io.ReadFull(reader, got); err != nil || string(got) != "websocket-frame" {
		t.Fatal("upgrade bytes lost", err)
	}
	p.Close()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("HTTP upgrade left alive after shutdown")
	}
}

func TestClientProxyHTTPSStillValidatesCertificates(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	template := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		DNSNames: []string{"phone.example.ts.net"}, IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, _ := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	cert, _ := x509.ParseCertificate(der)
	roots := x509.NewCertPool()
	roots.AddCert(cert)
	upstream := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Proxy-Authorization") != "" {
			t.Error("local credentials leaked through TLS")
		}
		io.WriteString(w, "encrypted-server-response")
	}))
	upstream.TLS = &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}}
	upstream.StartTLS()
	defer upstream.Close()
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, strings.TrimPrefix(upstream.URL, "https://"))
	}
	p, _ := newClientProxy("phone.example.ts.net:443", testProxyPassword, testProxyRealm, dial, dial)
	defer p.Close()
	s := httptest.NewServer(p)
	defer s.Close()
	proxyURL, _ := url.Parse(s.URL)
	proxyURL.User = url.UserPassword("pocketrisu", testProxyPassword)
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL), TLSClientConfig: &tls.Config{RootCAs: roots}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}
	resp, err := client.Get("https://phone.example.ts.net/test")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "encrypted-server-response" {
		t.Fatal("TLS response lost")
	}
	badTransport := &http.Transport{Proxy: http.ProxyURL(proxyURL), TLSClientConfig: &tls.Config{RootCAs: x509.NewCertPool()}}
	defer badTransport.CloseIdleConnections()
	if _, err = (&http.Client{Transport: badTransport}).Get("https://phone.example.ts.net/test"); err == nil {
		t.Fatal("untrusted certificate was accepted")
	}
}

func TestClientProxyAuthAndOrigin(t *testing.T) {
	var tail, direct atomic.Int32
	dial := func(counter *atomic.Int32) proxyDial {
		return func(context.Context, string, string) (net.Conn, error) {
			counter.Add(1)
			return nil, fmt.Errorf("test dial blocked")
		}
	}
	p, err := newClientProxy("phone.example.ts.net:443", testProxyPassword, testProxyRealm, dial(&tail), dial(&direct))
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	for _, auth := range []string{"", "Basic wrong"} {
		r := httptest.NewRequest("CONNECT", "http://phone.example.ts.net:443", nil)
		r.Host = "phone.example.ts.net:443"
		r.Header.Set("Proxy-Authorization", auth)
		w := httptest.NewRecorder()
		p.ServeHTTP(w, r)
		if w.Code != 407 || w.Header().Get("Proxy-Authenticate") == "" {
			t.Fatal("missing local authentication challenge")
		}
	}
	if tail.Load() != 0 || direct.Load() != 0 {
		t.Fatal("unauthenticated request reached network")
	}
	for _, host := range []string{"other.example.ts.net:443", "100.64.0.2:443", "[fd7a:115c:a1e0::2]:443"} {
		if _, err := p.dial(context.Background(), "tcp", host); err == nil {
			t.Fatal("foreign tailnet destination accepted")
		}
	}
	if tail.Load() != 0 || direct.Load() != 0 {
		t.Fatal("foreign tailnet destination dialed")
	}
	p.dial(context.Background(), "tcp", "PHONE.EXAMPLE.TS.NET.:0443")
	p.dial(context.Background(), "tcp", "public.example:443")
	if tail.Load() != 1 || direct.Load() != 1 {
		t.Fatal("wrong network selected")
	}
	for _, target := range []string{"phone.ts.net:0", "phone.ts.net:65536", "phone.ts.net:abc", "phone.ts.net:443/path", "public.example:443"} {
		if candidate, err := newClientProxy(target, testProxyPassword, testProxyRealm, dial(&tail), dial(&direct)); err == nil {
			candidate.Close()
			t.Fatalf("bad target accepted: %s", target)
		}
	}
}

func TestClientProxyConnectStreamsAndCloses(t *testing.T) {
	remoteClosed := make(chan struct{})
	dial := func(context.Context, string, string) (net.Conn, error) {
		a, b := net.Pipe()
		go func() { defer close(remoteClosed); defer b.Close(); io.Copy(b, b) }()
		return a, nil
	}
	p, err := newClientProxy("phone.example.ts.net:443", testProxyPassword, testProxyRealm, dial, dial)
	if err != nil {
		t.Fatal(err)
	}
	s := httptest.NewServer(p)
	defer s.Close()
	defer p.Close()
	c, err := net.Dial("tcp", strings.TrimPrefix(s.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(5 * time.Second))
	fmt.Fprintf(c, "CONNECT phone.example.ts.net:443 HTTP/1.1\r\nHost: phone.example.ts.net:443\r\nProxy-Authorization: %s\r\n\r\n", testAuth())
	r := bufio.NewReader(c)
	response, err := http.ReadResponse(r, &http.Request{Method: "CONNECT"})
	if err != nil || response.StatusCode != 200 {
		t.Fatalf("CONNECT: %v %v", response, err)
	}
	// Represents TLS handshake/stream bytes. The proxy must not terminate TLS.
	want := []byte{22, 3, 3, 0, 4, 1, 2, 3, 4}
	if _, err := c.Write(want); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(want))
	if _, err := io.ReadFull(r, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatal("tunnel modified bytes")
	}
	p.Close()
	select {
	case <-remoteClosed:
	case <-time.After(time.Second):
		t.Fatal("shutdown left a tunnel alive")
	}
}

func TestClientProxyHTTPAndNativeDownload(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Proxy-Authorization") != "" || r.Header.Get("Proxy-Connection") != "" || r.Header.Get("X-PocketRisu-Download") != "" {
			t.Error("proxy credentials leaked")
		}
		if r.Header.Get("Cookie") != "synthetic-cookie" {
			t.Error("cookie lost")
		}
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/octet-stream")
		fmt.Fprintf(w, "%s:%s:%s", r.Method, r.URL.RawQuery, string(body))
	}))
	defer upstream.Close()
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, strings.TrimPrefix(upstream.URL, "http://"))
	}
	p, _ := newClientProxy("phone.example.ts.net:80", testProxyPassword, testProxyRealm, dial, dial)
	defer p.Close()
	s := httptest.NewServer(p)
	defer s.Close()
	proxyURL, _ := url.Parse(s.URL)
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	r, _ := http.NewRequest("POST", "http://phone.example.ts.net/api/test?q=synthetic", strings.NewReader("stream-data"))
	r.Header.Set("Proxy-Authorization", testAuth())
	r.Header.Set("Proxy-Connection", "keep-alive")
	r.Header.Set("Cookie", "synthetic-cookie")
	resp, err := client.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "POST:q=synthetic:stream-data" {
		t.Fatalf("request lost data: %s", body)
	}
	r, _ = http.NewRequest("GET", "http://public.example/native-download", nil)
	r.Header.Set("Proxy-Authorization", testAuth())
	r.Header.Set("X-PocketRisu-Download", "http://phone.example.ts.net/file")
	resp, err = client.Do(r)
	if err != nil { t.Fatal(err) }
	resp.Body.Close()
	if resp.StatusCode != 403 { t.Fatal("web request was allowed to change origin using the native download header") }
	for _, target := range []string{"http://phone.example.ts.net/file?q=download", "http://other.example.ts.net/file"} {
		r, _ = http.NewRequest("GET", s.URL+"/native-download", nil)
		r.Header.Set("Proxy-Authorization", testAuth())
		r.Header.Set("X-PocketRisu-Download", target)
		r.Header.Set("Cookie", "synthetic-cookie")
		resp, err = http.DefaultClient.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		if strings.Contains(target, "other.") {
			if resp.StatusCode != 403 {
				t.Fatal("foreign native download allowed")
			}
		} else if string(body) != "GET:q=download:" {
			t.Fatalf("download lost data: %s", body)
		}
	}
}
