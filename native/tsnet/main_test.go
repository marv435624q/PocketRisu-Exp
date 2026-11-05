package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestProxyPreservesAuthenticationAndOrigin(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("risu-auth") != "existing-jwt" || r.Header.Get("Authorization") != "Bearer model-key" || r.Header.Get("Cookie") != "session=existing" {
			t.Error("authentication headers changed")
		}
		if r.Host != "pocketrisu.example.ts.net" || r.Header.Get("Origin") != "https://pocketrisu.example.ts.net" {
			t.Error("public origin/host changed")
		}
		if r.URL.RequestURI() != "/api/asset/test?x=1" {
			t.Error("path/query changed")
		}
		if strings.Contains(r.Header.Get("X-Forwarded-For"), "forged") {
			t.Error("forwarded header spoof retained")
		}
		w.Header().Set("Set-Cookie", "session=next; HttpOnly; Secure")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, "auth required")
	}))
	defer upstream.Close()
	u, _ := url.Parse(upstream.URL)
	proxy := httptest.NewServer(newProxy(u))
	defer proxy.Close()
	r, _ := http.NewRequest("GET", proxy.URL+"/api/asset/test?x=1", nil)
	r.Host = "pocketrisu.example.ts.net"
	r.Header.Set("risu-auth", "existing-jwt")
	r.Header.Set("Authorization", "Bearer model-key")
	r.Header.Set("Cookie", "session=existing")
	r.Header.Set("Origin", "https://pocketrisu.example.ts.net")
	r.Header.Set("X-Forwarded-For", "forged")
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 401 || resp.Header.Get("Set-Cookie") == "" {
		t.Fatal("upstream auth response changed")
	}
}

func TestProxyFlushesStreamingEvents(t *testing.T) {
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: first\n\n")
		w.(http.Flusher).Flush()
		<-release
		_, _ = io.WriteString(w, "data: last\n\n")
	}))
	defer upstream.Close()
	u, _ := url.Parse(upstream.URL)
	proxy := httptest.NewServer(newProxy(u))
	defer proxy.Close()
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(proxy.URL)
	if err != nil {
		close(release)
		t.Fatal(err)
	}
	defer resp.Body.Close()
	first, err := bufio.NewReader(resp.Body).ReadString('\n')
	close(release)
	if err != nil || first != "data: first\n" {
		t.Fatalf("stream buffered: %q %v", first, err)
	}
}

func TestProxyWebsocketUpgradeAndBytes(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("risu-auth") != "ws-jwt" {
			t.Error("websocket auth lost")
		}
		conn, rw, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		_, _ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
		_ = rw.Flush()
		payload := make([]byte, 4)
		_, err = io.ReadFull(rw, payload)
		if err == nil {
			_, _ = rw.Write(payload)
			_ = rw.Flush()
		}
	}))
	defer upstream.Close()
	u, _ := url.Parse(upstream.URL)
	proxy := httptest.NewServer(newProxy(u))
	defer proxy.Close()
	p, _ := url.Parse(proxy.URL)
	conn, err := net.Dial("tcp", p.Host)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	_, _ = fmt.Fprintf(conn, "GET /ws HTTP/1.1\r\nHost: %s\r\nConnection: Upgrade\r\nUpgrade: websocket\r\nrisu-auth: ws-jwt\r\n\r\n", p.Host)
	r := bufio.NewReader(conn)
	resp, err := http.ReadResponse(r, nil)
	if err != nil || resp.StatusCode != 101 {
		t.Fatalf("upgrade: %v %v", resp, err)
	}
	_, _ = conn.Write([]byte("ping"))
	got := make([]byte, 4)
	_, err = io.ReadFull(r, got)
	if err != nil || string(got) != "ping" {
		t.Fatalf("duplex forwarding: %q %v", got, err)
	}
}
