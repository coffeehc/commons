package httpclient

import (
	"crypto/tls"
	"time"
	"net"
)

type DialFunc func(addr string) (net.Conn, error)

type HttpClientOption struct {
	Name                string
	Dial                DialFunc
	DialDualStack       bool
	TLSConfig           *tls.Config
	MaxConnsPerHost     int
	MaxIdleConnDuration time.Duration
	ReadBufferSize      int
	WriteBufferSize     int
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	MaxResponseBodySize int
	//DisableHeaderNamesNormalizing bool // true
}

func (this *HttpClientOption)GetName() string {
	if this.Name == "" {
		this.Name = "http client"
	}
	return this.Name
}

func (this *HttpClientOption)GetDial() func(addr string) (net.Conn, error) {
	return this.Dial
}

func (this *HttpClientOption)GetDialDualStack() bool {
	return this.DialDualStack
}

func (this *HttpClientOption)GetTLSConfig() *tls.Config {
	return this.TLSConfig
}

func (this *HttpClientOption)GetMaxConnsPerHost() int {
	if this.MaxConnsPerHost == 0 {
		this.MaxConnsPerHost = 1000
	}
	return this.MaxConnsPerHost
}
func (this *HttpClientOption)GetMaxIdleConnDuration() time.Duration {
	if this.MaxIdleConnDuration == 0 {
		this.MaxIdleConnDuration = time.Second * 30
	}
	return this.MaxIdleConnDuration
}
func (this *HttpClientOption)GetReadBufferSize() int {
	if this.ReadBufferSize == 0 {
		this.ReadBufferSize = 1024 * 1024
	}
	return this.ReadBufferSize
}
func (this *HttpClientOption)GetWriteBufferSize() int {
	if this.WriteBufferSize == 0 {
		this.WriteBufferSize = 1024 * 1024
	}
	return this.WriteBufferSize
}
func (this *HttpClientOption)GetReadTimeout() time.Duration {
	if this.ReadTimeout == 0 {
		this.ReadTimeout = time.Second * 30
	}
	return this.ReadTimeout
}
func (this *HttpClientOption)GetWriteTimeout() time.Duration {
	if this.WriteTimeout == 0 {
		this.WriteTimeout = time.Second * 30
	}
	return this.WriteTimeout
}
func (this *HttpClientOption)GetMaxResponseBodySize() int {
	if this.MaxResponseBodySize == 0 {
		this.MaxResponseBodySize = 4 * 1024
	}
	return this.MaxResponseBodySize
}

