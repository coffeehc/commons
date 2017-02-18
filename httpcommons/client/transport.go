package client

import (
	"net"
	"net/http"
	"time"
)

type TransportBuilder func() *http.Transport

var _TransportBuilder = func() *http.Transport {
	return &http.Transport{}
}

func NewTimeoutTransportBuilder(transportBuilder TransportBuilder, timeout time.Duration, keepAlive time.Duration) TransportBuilder {
	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: keepAlive,
	}
	return func() *http.Transport {
		transport := buildTransport(transportBuilder)
		transport.DialContext = dialer.DialContext
		return transport
	}
}

func NewProxyTransportBuilder(transportBuilder TransportBuilder, proxyGetter ProxyGetter) TransportBuilder {
	p := newProxy(proxyGetter)
	return func() *http.Transport {
		transport := buildTransport(transportBuilder)
		transport.Proxy = p.proxyFromEnvironment
		return transport
	}
}

func buildTransport(transportBuilder TransportBuilder) *http.Transport {
	if transportBuilder == nil {
		transportBuilder = _TransportBuilder
	}
	transport := transportBuilder()
	return transport

}
