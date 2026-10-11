package httpc

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"

	"go.uber.org/zap"
)

func TestDefaultTLSRejectsUnknownAuthorityAndExplicitCAWorks(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer server.Close()
	for _, transport := range []http.RoundTripper{BuildTransport(), BuildDNSCacheTransport()} {
		tlsConfig := transport.(*http.Transport).TLSClientConfig
		if tlsConfig.InsecureSkipVerify || tlsConfig.MinVersion < tls.VersionTLS12 {
			t.Fatal("insecure transport default")
		}
		transport.(*http.Transport).CloseIdleConnections()
	}
	client := NewClient(zap.NewNop())
	defer client.GetClient().CloseIdleConnections()
	if _, err := client.R().Get(server.URL); err == nil {
		t.Fatal("unknown CA accepted")
	}
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	client.SetTLSClientConfig(&tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12})
	response, err := client.R().Get(server.URL)
	if err != nil || response.StatusCode() != http.StatusOK {
		t.Fatalf("explicit test CA rejected: %v", err)
	}
}

func TestInjectedCookieJarSurvivesInitialization(t *testing.T) {
	jar, _ := cookiejar.New(nil)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "fixture", Value: "public-example", Path: "/"})
	}))
	defer server.Close()
	client := NewClientWithCookieJar(jar, zap.NewNop())
	defer client.GetClient().CloseIdleConnections()
	if client.GetClient().Jar != jar {
		t.Fatal("injected jar replaced")
	}
	if _, err := client.R().Get(server.URL); err != nil {
		t.Fatal(err)
	}
	address, _ := url.Parse(server.URL)
	if cookies := jar.Cookies(address); len(cookies) != 1 || cookies[0].Value != "public-example" {
		t.Fatal("server cookie not retained by injected jar")
	}
}

func TestConstructorDoesNotAutomaticallyRetryMutation(t *testing.T) {
	client := NewClient(nil)
	defer client.GetClient().CloseIdleConnections()
	if client.RetryCount != 0 {
		t.Fatal("unsafe automatic retries")
	}
	client.SetRetryCount(2)
	if client.RetryCount != 2 {
		t.Fatal("explicit caller retry policy unavailable")
	}
}
