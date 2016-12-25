package httpclient

import (
	"crypto/tls"
	"net"
	"time"
)

//DialFunc 连接函数
type DialFunc func(addr string) (net.Conn, error)

//Option http clien's option
type Option struct {
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

//GetName 获取 client 的名称,默认为 http client
func (option *Option) GetName() string {
	if option.Name == "" {
		option.Name = "http client"
	}
	return option.Name
}

//GetDial return the Dial func in Option internal
func (option *Option) GetDial() func(addr string) (net.Conn, error) {
	return option.Dial
}

//GetDialDualStack Get DialDualStack value
func (option *Option) GetDialDualStack() bool {
	return option.DialDualStack
}

//GetTLSConfig Get TLSConfig
func (option *Option) GetTLSConfig() *tls.Config {
	return option.TLSConfig
}

//GetMaxConnsPerHost 限制每个 Host 的最大并发连接数,默认为1000
func (option *Option) GetMaxConnsPerHost() int {
	if option.MaxConnsPerHost == 0 {
		option.MaxConnsPerHost = 1000
	}
	return option.MaxConnsPerHost
}

//GetMaxIdleConnDuration 最大的空闲连接时间,默认30秒
func (option *Option) GetMaxIdleConnDuration() time.Duration {
	if option.MaxIdleConnDuration == 0 {
		option.MaxIdleConnDuration = time.Second * 30
	}
	return option.MaxIdleConnDuration
}

//GetReadBufferSize 最大的 Read Buf大小,默认为1M
func (option *Option) GetReadBufferSize() int {
	if option.ReadBufferSize == 0 {
		option.ReadBufferSize = 1024 * 1024
	}
	return option.ReadBufferSize
}

//GetWriteBufferSize 最大的 Write Buf大小,默认为1M
func (option *Option) GetWriteBufferSize() int {
	if option.WriteBufferSize == 0 {
		option.WriteBufferSize = 1024 * 1024
	}
	return option.WriteBufferSize
}

//GetReadTimeout 读超时,默认30秒
func (option *Option) GetReadTimeout() time.Duration {
	if option.ReadTimeout == 0 {
		option.ReadTimeout = time.Second * 30
	}
	return option.ReadTimeout
}

//GetWriteTimeout 写超时,默认30秒
func (option *Option) GetWriteTimeout() time.Duration {
	if option.WriteTimeout == 0 {
		option.WriteTimeout = time.Second * 30
	}
	return option.WriteTimeout
}

//GetMaxResponseBodySize 最大的 Response body 的大小
func (option *Option) GetMaxResponseBodySize() int {
	if option.MaxResponseBodySize == 0 {
		option.MaxResponseBodySize = 4 * 1024 * 1024
	}
	return option.MaxResponseBodySize
}
