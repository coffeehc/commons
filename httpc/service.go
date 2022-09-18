package httpc

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

func NewClient(logger *zap.Logger) *resty.Client {
	httpClient := resty.New()
	ClientInitSetting(httpClient, logger)
	httpClient.SetTransport(&http.Transport{
		Proxy: http.ProxyFromEnvironment,
		//DialContext:           DnsCacheDialContext(GetDialer()),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          1000,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   30 * time.Second,
		ExpectContinueTimeout: 0,
	})
	//httpClient.SetPreRequestHook(func(client *resty.Client, request *http.Request) error {
	//	DefaultLimiter.Take()
	//	return nil
	//})
	return httpClient
}

func ClientInitSetting(httpClient *resty.Client, logger *zap.Logger) {
	httpClient.RemoveProxy()
	httpClient.SetLogger(logger.Sugar())
	httpClient.SetRetryCount(3)
	httpClient.Header.Set("user-agent", "coffee's")
	httpClient.SetTimeout(time.Second * 15)
	httpClient.SetCloseConnection(true)
	httpClient.SetDisableWarn(true)
	httpClient.SetRetryMaxWaitTime(time.Second)
	httpClient.SetAllowGetMethodPayload(true)
	httpClient.SetCookieJar(NewNullCookieJar())
}

func NewClientWithCookieJar(cookieJar http.CookieJar, logger *zap.Logger) *resty.Client {
	httpClient := resty.NewWithClient(&http.Client{Jar: cookieJar})
	ClientInitSetting(httpClient, logger)
	httpClient.SetTransport(&http.Transport{
		Proxy: http.ProxyFromEnvironment,
		//DialContext:           DnsCacheDialContext(GetDialer()),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 0,
	})
	//httpClient.SetPreRequestHook(func(client *resty.Client, request *http.Request) error {
	//	DefaultLimiter.Take()
	//	return nil
	//})
	return httpClient
}

//var DefaultLimiter = ratelimit.New(200, ratelimit.WithSlack(50))

func BuildDNSCacheTransport() http.RoundTripper {
	roundTripper := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		//DialContext:           DnsCacheDialContext(GetDialer()),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 0,
	}
	return roundTripper
}

var defaultResolver = NewResolver(time.Minute * 10)

func GetDialer() *net.Dialer {
	dialer := &net.Dialer{
		Timeout:   60 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	return dialer
}

func DnsCacheDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network string, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, _ := defaultResolver.Get(ctx, host) // 这里自己实现了一个带缓存的Resolver，但是这个Resolver没有识别unix socket的功能，如果host里有port也不能识别，所以host不能带port
		for _, ip := range ips {
			conn, err := dialer.DialContext(ctx, network, ip+":"+port) // 这里我们已经解析出来了ip和port，那么net.Dialer判断出来是个ip就不会再去解析了
			if err == nil {
				return conn, nil
			}
		}
		return dialer.DialContext(ctx, network, address) // //如果前面解析失败了就老老实实用原address去调用吧，可能address是个unix socket呢。这里是个兜底，前面都失败了那么我们还可以用原来的方式去做
	}
}

func defaultTransportDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return dialer.DialContext
}
