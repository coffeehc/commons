package httpclient

import (
	"errors"
	"github.com/valyala/fasthttp"
	"time"
)

type _HttpClient struct {
	client *fasthttp.Client
}

func (httpClient *_HttpClient) Get(dst []byte, url string) (statusCode int, body []byte, err error) {
	return httpClient.client.Get(dst, url)
}
func (httpClient *_HttpClient) GetTimeout(dst []byte, url string, timeout time.Duration) (statusCode int, body []byte, err error) {
	return httpClient.client.GetTimeout(dst, url, timeout)
}
func (httpClient *_HttpClient) GetDeadline(dst []byte, url string, deadline time.Time) (statusCode int, body []byte, err error) {
	return httpClient.client.GetDeadline(dst, url, deadline)
}
func (httpClient *_HttpClient) Post(dst []byte, url string, postArgs Args) (statusCode int, body []byte, err error) {
	var args *fasthttp.Args
	if postArgs != nil {
		args = postArgs.getFastHTTPArgs()
	}
	return httpClient.client.Post(dst, url, args)
}
func (httpClient *_HttpClient) DoTimeout(req Request, resp Response, timeout time.Duration) error {
	if req == nil {
		return errors.New("Request 不能为 nil")
	}
	if resp == nil {
		return errors.New("Request 不能为 nil")
	}
	return httpClient.client.DoTimeout(req.getFastHTTPRequest(), resp.getFastHTTPResponse(), timeout)
}
func (httpClient *_HttpClient) DoDeadline(req Request, resp Response, deadline time.Time) error {
	if req == nil {
		return errors.New("Request 不能为 nil")
	}
	if resp == nil {
		return errors.New("Request 不能为 nil")
	}
	return httpClient.client.DoDeadline(req.getFastHTTPRequest(), resp.getFastHTTPResponse(), deadline)
}
func (httpClient *_HttpClient) Do(req Request, resp Response) error {
	if req == nil {
		return errors.New("Request 不能为 nil")
	}
	if resp == nil {
		return errors.New("Request 不能为 nil")
	}
	return httpClient.client.Do(req.getFastHTTPRequest(), resp.getFastHTTPResponse())
}
