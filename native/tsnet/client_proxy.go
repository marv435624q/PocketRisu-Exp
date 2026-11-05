package main

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type proxyDial func(context.Context, string, string) (net.Conn, error)

// Only the configured origin may use the embedded tailnet identity. Ordinary
// internet requests retain the device network; arbitrary other tailnet nodes
// are never reachable through this local proxy.
type clientProxy struct {
	target, auth, realm  string
	tailDial, directDial proxyDial
	transport            *http.Transport
	mu                   sync.Mutex
	tunnels              map[net.Conn]struct{}
	upgrades             map[*proxyUpgrade]struct{}
	closed               bool
}

func normalizeProxyAddress(address string) (string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" || port == "" {
		return "", errors.New("invalid proxy destination")
	}
	u, err := url.Parse("http://" + net.JoinHostPort(host, port))
	if err != nil || u.Port() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("invalid proxy destination")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", errors.New("invalid proxy port")
	}
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if ip, err := netip.ParseAddr(host); err == nil {
		host = ip.String()
	}
	return net.JoinHostPort(host, strconv.Itoa(n)), nil
}

func isTailnetHost(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(strings.Trim(host, "[]")), ".")
	if strings.HasSuffix(host, ".ts.net") {
		return true
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && (netip.MustParsePrefix("100.64.0.0/10").Contains(ip) || netip.MustParsePrefix("fd7a:115c:a1e0::/48").Contains(ip))
}

func newClientProxy(target, password, realm string, tail, direct proxyDial) (*clientProxy, error) {
	target, err := normalizeProxyAddress(target)
	if err != nil {
		return nil, err
	}
	host, _, _ := net.SplitHostPort(target)
	if !isTailnetHost(host) || len(password) < 32 || len(realm) < 16 || strings.ContainsAny(realm, "\r\n\"") {
		return nil, errors.New("invalid client proxy configuration")
	}
	p := &clientProxy{target: target, auth: "Basic " + base64.StdEncoding.EncodeToString([]byte("pocketrisu:"+password)), realm: realm,
		tailDial: tail, directDial: direct, tunnels: make(map[net.Conn]struct{}), upgrades: make(map[*proxyUpgrade]struct{})}
	p.transport = &http.Transport{Proxy: nil, DialContext: p.dial, ForceAttemptHTTP2: false, TLSHandshakeTimeout: 30 * time.Second, IdleConnTimeout: 90 * time.Second}
	return p, nil
}

func (p *clientProxy) dial(ctx context.Context, network, address string) (net.Conn, error) {
	address, err := normalizeProxyAddress(address)
	if err != nil {
		return nil, err
	}
	if address == p.target {
		return p.tailDial(ctx, network, address)
	}
	host, _, _ := net.SplitHostPort(address)
	if isTailnetHost(host) {
		return nil, errors.New("tailnet destination is not the configured server")
	}
	return p.directDial(ctx, network, address)
}

func (p *clientProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("Proxy-Authorization")), []byte(p.auth)) != 1 {
		w.Header().Set("Proxy-Authenticate", `Basic realm="`+p.realm+`"`)
		w.Header().Set("Connection", "close")
		http.Error(w, "Local proxy authentication required", http.StatusProxyAuthRequired)
		return
	}
	r.Header.Del("Proxy-Authorization")
	r.Header.Del("Proxy-Connection")
	if r.Method == http.MethodConnect {
		p.connect(w, r)
		return
	}
	// Native downloads cannot use DownloadManager's system resolver. Their
	// authenticated local request uses this same transport, with end-to-end TLS.
	if target := r.Header.Get("X-PocketRisu-Download"); target != "" {
		r.Header.Del("X-PocketRisu-Download")
		// A webpage's authenticated forward-proxy request must never become a
		// native download and change origin. Only HttpURLConnection's local,
		// origin-form endpoint may select this transport.
		if r.URL.IsAbs() || r.URL.Path != "/native-download" {
			http.Error(w, "Native download endpoint required", 403)
			return
		}
		u, err := url.Parse(target)
		if err != nil || u.User != nil || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") {
			http.Error(w, "Invalid download", 400)
			return
		}
		port := u.Port()
		if port == "" {
			if u.Scheme == "https" {
				port = "443"
			} else {
				port = "80"
			}
		}
		address, err := normalizeProxyAddress(net.JoinHostPort(u.Hostname(), port))
		if err != nil || address != p.target || r.Method != "GET" {
			http.Error(w, "Download origin is not the configured server", 403)
			return
		}
		r.URL = u
		r.Host = u.Host
	} else if r.URL.Scheme != "http" || r.URL.Host == "" || r.URL.User != nil {
		http.Error(w, "Absolute HTTP URL or CONNECT required", 400)
		return
	}
	proxy := &httputil.ReverseProxy{Rewrite: func(pr *httputil.ProxyRequest) {
		pr.Out.URL = r.URL
		pr.Out.Host = r.Host
		// No Forwarded/Proxy-Authorization or other local credentials go upstream.
		pr.Out.Header.Del("Proxy-Authorization")
		pr.Out.Header.Del("Proxy-Connection")
	}, Transport: p, FlushInterval: -1,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) { http.Error(w, "Remote connection failed", 502) }}
	proxy.ServeHTTP(w, r)
}

// ReverseProxy hijacks HTTP WebSockets too; http.Server.Close alone does not
// close those connections. Track the upgraded upstream until its body closes.
type proxyUpgrade struct {
	io.ReadWriteCloser
	owner *clientProxy
	once  sync.Once
}

func (u *proxyUpgrade) Close() (err error) {
	u.once.Do(func() {
		err = u.ReadWriteCloser.Close()
		u.owner.mu.Lock()
		delete(u.owner.upgrades, u)
		u.owner.mu.Unlock()
	})
	return
}
func (p *clientProxy) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := p.transport.RoundTrip(r)
	if err != nil || response.StatusCode != http.StatusSwitchingProtocols {
		return response, err
	}
	if body, ok := response.Body.(io.ReadWriteCloser); ok {
		u := &proxyUpgrade{ReadWriteCloser: body, owner: p}
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			body.Close()
			return nil, errors.New("client proxy stopped")
		}
		p.upgrades[u] = struct{}{}
		p.mu.Unlock()
		response.Body = u
	}
	return response, nil
}

func (p *clientProxy) connect(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	remote, err := p.dial(ctx, "tcp", r.Host)
	cancel()
	if err != nil {
		http.Error(w, "Remote connection failed", 502)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		remote.Close()
		http.Error(w, "Tunnel unavailable", 500)
		return
	}
	local, buffered, err := hj.Hijack()
	if err != nil {
		remote.Close()
		return
	}
	_ = local.SetDeadline(time.Time{}) // Header timeouts must not limit long streams.
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		local.Close()
		remote.Close()
		return
	}
	p.tunnels[local] = struct{}{}
	p.tunnels[remote] = struct{}{}
	p.mu.Unlock()
	defer func() {
		local.Close()
		remote.Close()
		p.mu.Lock()
		delete(p.tunnels, local)
		delete(p.tunnels, remote)
		p.mu.Unlock()
	}()
	if _, err = buffered.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
		return
	}
	if err = buffered.Flush(); err != nil {
		return
	}
	// Keep bytes already read past CONNECT's headers (TLS clients can pipeline).
	finished := make(chan struct{}, 2)
	go func() {
		io.Copy(remote, buffered.Reader)
		if !closeWrite(remote) {
			remote.Close()
		}
		finished <- struct{}{}
	}()
	go func() {
		io.Copy(local, remote)
		if !closeWrite(local) {
			local.Close()
		}
		finished <- struct{}{}
	}()
	<-finished
	<-finished
}

func closeWrite(c net.Conn) bool {
	if v, ok := c.(interface{ CloseWrite() error }); ok {
		return v.CloseWrite() == nil
	}
	return false
}

func (p *clientProxy) Close() {
	p.mu.Lock()
	p.closed = true
	var all []io.Closer
	for c := range p.tunnels {
		all = append(all, c)
	}
	for c := range p.upgrades {
		all = append(all, c)
	}
	p.mu.Unlock()
	for _, c := range all {
		c.Close()
	}
	p.transport.CloseIdleConnections()
}
