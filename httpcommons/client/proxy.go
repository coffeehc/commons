package client

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type ProxyProvider interface {
	GetHTTPSProxy() string
	GetHTTPProxy() string
}

type AddrsProxyProvicer struct {
	HttpProxys  []string
	HttpsProxys []string
}

func (pg *AddrsProxyProvicer) GetHTTPSProxy() string {
	length := len(pg.HttpsProxys)
	if length == 0 {
		return ""
	}
	if length == 1 {
		return pg.HttpsProxys[0]
	}
	return pg.HttpsProxys[rand.Int31n(int32(length))]
}
func (pg *AddrsProxyProvicer) GetHTTPProxy() string {
	length := len(pg.HttpProxys)
	if length == 0 {
		return ""
	}
	if length == 1 {
		return pg.HttpProxys[0]
	}
	return pg.HttpProxys[rand.Int31n(int32(length))]
}

func NewProxyByAddrProviter(provider ProxyProvider) (func(req *http.Request) (*url.URL, error), error) {
	if provider == nil {
		return nil, errors.New("ProxyProvider is nil")
	}
	p := &proxy{provider: provider}
	return p.proxyFromGetter, nil
}

type proxy struct {
	provider ProxyProvider
}

func (p *proxy) proxyFromGetter(req *http.Request) (*url.URL, error) {
	var proxy string
	if req.URL.Scheme == "https" {
		proxy = p.provider.GetHTTPSProxy()
	}
	if proxy == "" {
		proxy = p.provider.GetHTTPProxy()
		if proxy != "" && os.Getenv("REQUEST_METHOD") != "" {
			return nil, errors.New("net/http: refusing to use HTTP_PROXY value in CGI environment; see golang.org/s/cgihttpproxy")
		}
	}
	if proxy == "" {
		return nil, nil
	}
	if !useProxy(canonicalAddr(req.URL)) {
		return nil, nil
	}
	proxyURL, err := url.Parse(proxy)
	if err != nil || !strings.HasPrefix(proxyURL.Scheme, "http") {
		// proxy was bogus. Try prepending "http://" to it and
		// see if that parses correctly. If not, we fall
		// through and complain about the original one.
		if proxyURL, err := url.Parse("http://" + proxy); err == nil {
			return proxyURL, nil
		}
	}
	if err != nil {
		return nil, fmt.Errorf("invalid proxy address %q: %v", proxy, err)
	}
	return proxyURL, nil
}

var portMap = map[string]string{
	"http":  "80",
	"https": "443",
}

// canonicalAddr returns url.Host but always with a ":port" suffix
func canonicalAddr(url *url.URL) string {
	addr := url.Host
	if !hasPort(addr) {
		return addr + ":" + portMap[url.Scheme]
	}
	return addr
}

func hasPort(s string) bool { return strings.LastIndex(s, ":") > strings.LastIndex(s, "]") }

func useProxy(addr string) bool {
	if len(addr) == 0 {
		return true
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() {
			return false
		}
	}

	addr = strings.ToLower(strings.TrimSpace(addr))
	if hasPort(addr) {
		addr = addr[:strings.LastIndex(addr, ":")]
	}

	return true
}
