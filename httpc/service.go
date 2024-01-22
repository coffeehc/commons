package httpc

import (
	"context"
	"crypto/tls"
	"github.com/coffeehc/base/log"
	"net"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

var DefaultClient = NewClient(zap.NewNop())

var DefaultTransport = BuildTransport()

func BuildTransport() http.RoundTripper {
	return &http.Transport{
		DialContext:         DnsCacheDialContext(GetDialer()),
		DisableKeepAlives:   false,
		MaxIdleConnsPerHost: 5000,
		MaxConnsPerHost:     5000,
		//MaxIdleConns:          1000,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		ForceAttemptHTTP2: true,
	}
}

var EnableTrace = false

func NewClientNoProxy(logger *zap.Logger) *resty.Client {
	httpClient := resty.New()
	ClientInitSetting(httpClient, logger)
	httpClient.SetTransport(DefaultTransport)
	httpClient.SetCloseConnection(false)
	if EnableTrace {
		httpClient.EnableTrace()
	}
	return httpClient
}

func NewClient(logger *zap.Logger) *resty.Client {
	httpClient := resty.New()
	ClientInitSetting(httpClient, logger)
	httpClient.SetTransport(BuildTransport())
	httpClient.SetCloseConnection(false)
	if EnableTrace {
		httpClient.EnableTrace()
	}
	return httpClient
}

func ClientInitSetting(httpClient *resty.Client, logger *zap.Logger) {
	httpClient.RemoveProxy()
	httpClient.SetLogger(logger.Sugar())
	httpClient.SetRetryCount(3)
	httpClient.Header.Set("user-agent", "coffee's")
	httpClient.Header.Set("accept", "*/*")
	httpClient.SetTimeout(time.Second * 30)
	httpClient.SetCloseConnection(false)
	httpClient.SetDisableWarn(true)
	httpClient.SetRetryMaxWaitTime(time.Second)
	httpClient.SetAllowGetMethodPayload(true)
	httpClient.SetCookieJar(NewNullCookieJar())
}

func NewClientWithCookieJar(cookieJar http.CookieJar, logger *zap.Logger) *resty.Client {
	httpClient := resty.NewWithClient(&http.Client{Jar: cookieJar})
	if EnableTrace {
		httpClient.EnableTrace()
	}
	ClientInitSetting(httpClient, logger)
	httpClient.SetTransport(BuildTransport())
	httpClient.SetCloseConnection(false)
	return httpClient
}

//var DefaultLimiter = ratelimit.New(200, ratelimit.WithSlack(50))

func BuildDNSCacheTransport() http.RoundTripper {
	roundTripper := &http.Transport{
		DisableKeepAlives:     false,
		MaxIdleConnsPerHost:   5000,
		MaxConnsPerHost:       5000,
		IdleConnTimeout:       10 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 0,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		ForceAttemptHTTP2: true,
		//Proxy: http.ProxyFromEnvironment,
		DialContext: DnsCacheDialContext(GetDialer()),
	}
	return roundTripper
}

var defaultResolver = NewResolver(time.Minute*5, time.Minute*4)

func GetDialer() *net.Dialer {
	dialer := &net.Dialer{
		Timeout:   60 * time.Second,
		KeepAlive: 30 * time.Second,
		//FallbackDelay: 10 * time.Second,
		//DualStack: true,
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
		//if len(ips) > 0 {
		//	cryptos.GetRandInt(len(ips), 1)
		//}
		//log.Debug("获取缓存ip", zap.String("network", network), zap.String("host", host), zap.Strings("ips", ips))
		for _, ip := range ips {
			if ip == "" {
				continue
			}
			//log.Debug("命中DNS缓存", zap.String("network", network), zap.String("address", ip), zap.String("host", host))
			conn, err := dialer.DialContext(ctx, network, ip+":"+port) // 这里我们已经解析出来了ip和port，那么net.Dialer判断出来是个ip就不会再去解析了
			if err == nil {
				return conn, nil
			} else {
				log.Error("错误", zap.Error(err))
			}
		}
		//log.Debug("没有命中DNS缓存", zap.String("network", network), zap.String("address", address), zap.String("host", host))
		return dialer.DialContext(ctx, network, address) //如果前面解析失败了就老老实实用原address去调用吧，可能address是个unix socket呢。这里是个兜底，前面都失败了那么我们还可以用原来的方式去做
	}
}
