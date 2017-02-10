package client

import (
	"net"
	"net/http"
	"time"
)

type ClientOptions struct {
	UserAgent                      string
	Timeout                        time.Duration //对应从连接(Dial)到读完response body的整个时间(包含所有的redirect的时间)
	DialerTimeout                  time.Duration //限制建立TCP连接的时间
	DialerKeepAlive                time.Duration
	TransportTLSHandshakeTimeout   time.Duration //限制 TLS握手的时间
	TransportResponseHeaderTimeout time.Duration //限制读取response header的时间
	//TransportExpectContinueTimeout time.Duration //(可能引起不必要的风险,直接忽略这个值)限制client在发送包含 Expect: 100-continue的header到收到继续发送body的response之间的时间等待。注意在1.6中设置这个值会禁用HTTP/2(DefaultTransport自1.6.2起是个特例)
	TransportIdleConnTimeout time.Duration //控制连接池中一个连接可以idle多长时间
	TransportMaxIdleConns    int

	Transport *http.Transport
}

func (co *ClientOptions) GetUserAgent() string {
	if co.UserAgent == "" {
		co.UserAgent = "coffee client"
	}
	return co.UserAgent
}

func (co *ClientOptions) getTimeout() time.Duration {
	if co.Timeout == 0 {
		co.Timeout = 30 * time.Second
	}
	return co.Timeout
}
func (co *ClientOptions) getDialerTimeout() time.Duration {
	if co.DialerTimeout == 0 {
		co.DialerTimeout = 3 * time.Second
	}
	return co.DialerTimeout
}
func (co *ClientOptions) getDialerKeepAlive() time.Duration {
	if co.DialerKeepAlive == 0 {
		co.DialerKeepAlive = 60 * time.Second
	}
	return co.DialerKeepAlive
}
func (co *ClientOptions) getTransportTLSHandshakeTimeout() time.Duration {
	if co.TransportTLSHandshakeTimeout == 0 {
		co.TransportTLSHandshakeTimeout = 3 * time.Second
	}
	return co.TransportTLSHandshakeTimeout
}
func (co *ClientOptions) getTransportResponseHeaderTimeout() time.Duration {
	if co.TransportResponseHeaderTimeout == 0 {
		co.TransportResponseHeaderTimeout = 3 * time.Second
	}
	return co.TransportResponseHeaderTimeout
}

func (co *ClientOptions) getTransportIdleConnTimeout() time.Duration {
	if co.TransportIdleConnTimeout == 0 {
		co.TransportIdleConnTimeout = 90 * time.Second
	}
	return co.TransportIdleConnTimeout
}

func (co *ClientOptions) getTransportMaxIdleConns() int {
	if co.TransportMaxIdleConns == 0 {
		co.TransportMaxIdleConns = 1000
	}
	return co.TransportMaxIdleConns
}

func (co *ClientOptions) getTransport() *http.Transport {
	if co.Transport == nil {
		co.Transport = &http.Transport{
			//Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   co.getDialerTimeout(),
				KeepAlive: co.getDialerKeepAlive(),
			}).DialContext,
			MaxIdleConns:        co.getTransportMaxIdleConns(),
			IdleConnTimeout:     co.getTransportIdleConnTimeout(),
			TLSHandshakeTimeout: co.getTransportTLSHandshakeTimeout(),
			//ExpectContinueTimeout: 1 * time.Second,
		}
	}
	return co.Transport
}

func (co *ClientOptions) setClientOptions(c *http.Client) {
	if c.Transport == nil {
		c.Transport = co.getTransport()
	}
}
