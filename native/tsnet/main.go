// PocketRisu tsnet sidecar. stdin/stdout are private pipes owned by the Android
// service. Only the tailnet listener proxies HTTP; no public control listener.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	_ "tailscale.com/feature/androidbin"
	"tailscale.com/tsnet"
)

type snapshot struct {
	Type           string   `json:"type"`
	Status         string   `json:"status"`
	BackendState   string   `json:"backendState,omitempty"`
	Registered     *bool    `json:"registered,omitempty"`
	URL            string   `json:"url,omitempty"`
	AuthURL        string   `json:"authUrl,omitempty"`
	Error          string   `json:"error,omitempty"`
	Health         []string `json:"health,omitempty"`
	LogoutComplete bool     `json:"logoutComplete,omitempty"`
	ProxyPort      int      `json:"proxyPort,omitempty"`
}

func boolValue(v bool) *bool { return &v }

var outputMu sync.Mutex
var secretURL = regexp.MustCompile(`https?://[^\s"<>]+`)

func safeError(err error) string {
	if err == nil {
		return ""
	}
	return secretURL.ReplaceAllString(err.Error(), "[URL omitted]")
}

func emit(s snapshot) {
	s.Type = "status"
	outputMu.Lock()
	defer outputMu.Unlock()
	_ = json.NewEncoder(os.Stdout).Encode(s)
}

// Preserve cookies, Origin, risu-auth, Authorization and websocket upgrades.
// Rewrite drops untrusted Forwarded headers before installing our own values.
func newProxy(target *url.URL) *httputil.ReverseProxy {
	p := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.Host = r.In.Host
			r.SetXForwarded()
		},
		FlushInterval: -1,
		ErrorLog:      log.New(io.Discard, "", 0),
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, "PocketRisu localhost server is unavailable", http.StatusBadGateway)
		},
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil               // never send local traffic through inherited proxy vars
	transport.ResponseHeaderTimeout = 0 // long LLM requests and event streams
	p.Transport = transport
	return p
}

func run() error {
	options, err := parseRuntimeOptions(os.Args[1:], os.Stderr)
	if err != nil {
		return err
	}
	if options.checkArgs {
		// This branch shares the real parser and validation, but exits before
		// creating state, changing environment, starting tsnet or logging out.
		return json.NewEncoder(os.Stdout).Encode(struct {
			Type            string `json:"type"`
			Parsed          bool   `json:"parsed"`
			StateDirValid   bool   `json:"stateDirValid"`
			ClientMode      bool   `json:"clientMode"`
			LogoutRequested bool   `json:"logoutRequested"`
		}{"args-check", true, true, options.clientTarget != "", options.logoutOnly})
	}
	stateDir, hostname := &options.stateDir, &options.hostname
	logoutOnly, clientTarget := &options.logoutOnly, &options.clientTarget
	if options.requireRegistered {
		if err := requireExistingState(*stateDir); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(*stateDir, 0700); err != nil {
		return err
	}
	if err := os.Chmod(*stateDir, 0700); err != nil {
		return err
	}
	// Authentication is always interactive; don't inherit keys from host shells.
	clearInheritedTailscaleAuth(options.requireRegistered)
	_ = os.Setenv("TS_NO_LOGS_NO_SUPPORT", "true")
	srv := &tsnet.Server{Dir: *stateDir, Hostname: *hostname, RequireExistingRegistration: options.requireRegistered,
		Logf: func(string, ...any) {}, UserLogf: func(string, ...any) {}}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	var clientHTTP *http.Server
	var clientHandler *clientProxy
	var clientPort int
	if *clientTarget != "" && !*logoutOnly {
		password, realm := os.Getenv("POCKETRISU_PROXY_PASSWORD"), os.Getenv("POCKETRISU_PROXY_REALM")
		_ = os.Unsetenv("POCKETRISU_PROXY_PASSWORD")
		_ = os.Unsetenv("POCKETRISU_PROXY_REALM")
		var err error
		clientHandler, err = newClientProxy(*clientTarget, password, realm, srv.Dial, (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext)
		if err != nil {
			return err
		}
		ln, err := net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			return errors.New("client proxy listener failed")
		}
		clientPort = ln.Addr().(*net.TCPAddr).Port
		clientHTTP = &http.Server{Handler: clientHandler, ReadHeaderTimeout: 10 * time.Second, MaxHeaderBytes: 16 * 1024, IdleTimeout: 90 * time.Second,
			ErrorLog: log.New(io.Discard, "", 0)}
		defer func() { clientHTTP.Close(); clientHandler.Close() }()
		go func() {
			if err := clientHTTP.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
				cancel()
			}
		}()
		go func() { <-ctx.Done(); clientHTTP.Close(); clientHandler.Close() }()
	}
	// Parent death/pipe closure also releases the listener. No orphan daemon.
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			var cmd struct {
				Action string `json:"action"`
			}
			if json.Unmarshal(scanner.Bytes(), &cmd) == nil && cmd.Action == "stop" {
				cancel()
				return
			}
		}
		cancel()
	}()
	emit(snapshot{Status: "starting"})
	if err := srv.Start(); err != nil {
		return fmt.Errorf("tsnet.Start: %w", err)
	}
	defer srv.Close()
	lc, err := srv.LocalClient()
	if err != nil {
		return err
	}
	if *logoutOnly {
		logoutCtx, done := context.WithTimeout(ctx, 30*time.Second)
		err := lc.Logout(logoutCtx)
		done()
		if err != nil {
			return fmt.Errorf("logout failed; identity retained: %w", err)
		}
		// Close before removal, preventing late state writes. Only this fixed
		// service-owned directory is reset; save/, sqlite and logs are untouched.
		_ = srv.Close()
		if err := os.RemoveAll(*stateDir); err != nil {
			return err
		}
		emit(snapshot{Status: "off", LogoutComplete: true})
		return nil
	}
	var httpServer *http.Server
	var proxyURL string
	defer func() {
		if httpServer != nil {
			_ = httpServer.Close()
		}
	}()
	// Cancellation must unblock ListenTLS/Up and in-flight certificate requests.
	go func() { <-ctx.Done(); _ = srv.Close() }()
	loginRequested := false
	var retryTLS time.Time
	lastError := ""
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			emit(snapshot{Status: "off"})
			if options.verifyHTTPS {
				return errors.New("HTTPS verification canceled before success")
			}
			return nil
		}
		pollCtx, done := context.WithTimeout(ctx, 5*time.Second)
		st, err := lc.Status(pollCtx)
		done()
		if err != nil {
			emit(snapshot{Status: "reconnecting", Error: safeError(err)})
		} else {
			s := snapshot{Status: "connecting", BackendState: st.BackendState, Registered: boolValue(st.HaveNodeKey)}
			for _, h := range st.Health {
				s.Health = append(s.Health, safeError(errors.New(h)))
			}
			switch st.BackendState {
			case "NeedsLogin":
				if err := allowInteractiveLogin(options.requireRegistered); err != nil {
					return err
				}
				s.Status = "needs-login"
				s.AuthURL = st.AuthURL
				if !loginRequested {
					loginCtx, finish := context.WithTimeout(ctx, 5*time.Second)
					loginErr := lc.StartLoginInteractive(loginCtx)
					finish()
					loginRequested = loginErr == nil
					s.Error = safeError(loginErr)
				}
			case "NeedsMachineAuth":
				s.Status = "needs-approval"
			case "Running":
				loginRequested = false
				if clientHTTP != nil {
					s.Status = "running"
					s.ProxyPort = clientPort
					break
				}
				if httpServer == nil && !time.Now().Before(retryTLS) {
					retryTLS = time.Now().Add(30 * time.Second)
					if st.CurrentTailnet == nil || !st.CurrentTailnet.MagicDNSEnabled || len(st.CertDomains) == 0 {
						lastError = "Enable MagicDNS and HTTPS certificates in the Tailnet DNS settings"
					} else {
						domain := strings.TrimSuffix(st.CertDomains[0], ".")
						ln, listenErr := obtainHTTPSListener(ctx, domain, lc.CertPair, srv.ListenTLS)
						lastError = safeError(listenErr)
						if listenErr == nil {
							target, _ := url.Parse("http://127.0.0.1:6001")
							httpServer = &http.Server{Handler: newProxy(target), ReadHeaderTimeout: 15 * time.Second,
								IdleTimeout: 90 * time.Second, ErrorLog: log.New(io.Discard, "", 0)}
							proxyURL = "https://" + domain
							server := httpServer
							go func() {
								if serveErr := server.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) && ctx.Err() == nil {
									emit(snapshot{Status: "error", Registered: boolValue(true), Error: safeError(serveErr)})
									cancel()
								}
							}()
						}
					}
				}
				if httpServer != nil {
					s.Status = "running"
					s.URL = proxyURL
					if options.verifyHTTPS {
						probeCtx, finish := context.WithTimeout(ctx, 15*time.Second)
						result := verifyHTTPS(probeCtx, strings.TrimPrefix(proxyURL, "https://"), srv.Dial, nil)
						finish()
						_ = json.NewEncoder(os.Stdout).Encode(result)
						if !result.Success {
							return errors.New("HTTPS verification failed")
						}
						return nil
					}
				} else {
					s.Status = "https-error"
					s.Error = lastError
					if options.verifyHTTPS {
						emit(s)
						return errors.New("HTTPS certificate or listener verification failed")
					}
				}
			default:
				s.Status = "reconnecting"
			}
			emit(s)
		}
		select {
		case <-ctx.Done():
		case <-ticker.C:
		}
	}
}

func main() {
	rawArgs := os.Args
	args, diagnostic := normalizeStartupArgs(rawArgs, runtime.GOOS == "android", os.Getenv("TERMUX_EXEC__PROC_SELF_EXE"))
	os.Args = args
	if len(args) == 2 && args[1] == "--build-info" {
		info, err := currentBuildInformation()
		if err != nil {
			fmt.Fprintln(os.Stderr, "candidate build identity unavailable")
			os.Exit(1)
		}
		if err := json.NewEncoder(os.Stdout).Encode(info); err != nil {
			os.Exit(1)
		}
		return
	}
	if diagnosticRequested(rawArgs) {
		if err := json.NewEncoder(os.Stdout).Encode(diagnostic); err != nil {
			os.Exit(1)
		}
		return
	}
	// Keep the calling thread alive: PR_SET_PDEATHSIG is thread-specific.
	// Android force-stop or parent death must not leave either sidecar running.
	runtime.LockOSThread()
	parent := os.Getppid()
	if err := unix.Prctl(unix.PR_SET_PDEATHSIG, uintptr(syscall.SIGKILL), 0, 0, 0); err != nil {
		emit(snapshot{Status: "error", Error: "parent-death setup: " + safeError(err)})
		os.Exit(1)
	}
	if os.Getppid() != parent || parent == 1 {
		os.Exit(1)
	}
	if len(os.Args) >= 3 && os.Args[1] == "--exec-cloudflared" {
		self, err := os.Executable()
		payload := filepath.Clean(os.Args[2])
		if err != nil || !filepath.IsAbs(payload) || filepath.Dir(payload) != filepath.Dir(self) || filepath.Base(payload) != "libcloudflared_exec.so" {
			os.Exit(2)
		}
		if err := syscall.Exec(payload, append([]string{payload}, os.Args[3:]...), os.Environ()); err != nil {
			fmt.Fprintln(os.Stderr, "cloudflared exec:", safeError(err))
			os.Exit(1)
		}
	}
	if err := run(); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		if errors.Is(err, errArgumentParsing) {
			os.Exit(2)
		}
		emit(snapshot{Status: "error", Error: safeError(err)})
		os.Exit(1)
	}
}
