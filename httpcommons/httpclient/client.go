package httpclient

import (
	"errors"
	"time"

	"github.com/valyala/fasthttp"
)

type _HttpClient struct {
	client *fasthttp.Client
}

func (httpClient *_HttpClient) Get(url string) (statusCode int, body []byte, err error) {
	return httpClient.client.Get(nil, url)
}
func (httpClient *_HttpClient) GetTimeout(url string, timeout time.Duration) (statusCode int, body []byte, err error) {
	return httpClient.client.GetTimeout(nil, url, timeout)
}
func (httpClient *_HttpClient) GetDeadline(url string, deadline time.Time) (statusCode int, body []byte, err error) {
	return httpClient.client.GetDeadline(nil, url, deadline)
}
func (httpClient *_HttpClient) Post(url string, postArgs Args) (statusCode int, body []byte, err error) {
	var args *fasthttp.Args
	if postArgs != nil {
		args = postArgs.getFastHTTPArgs()
	}
	return httpClient.client.Post(nil, url, args)
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

func (httpClient *_HttpClient) DoTimeoutByContext(cxt ClientContext, timeout time.Duration) (int, []byte, error) {
	cxt.InjectRequestHeader()
	response := cxt.GetResponse()
	err := httpClient.DoTimeout(cxt.GetRequest(), response, timeout)
	if err != nil {
		return 0, nil, err
	}
	cxt.HandleResponseHeader()
	defer cxt.Release()
	return response.StatusCode(), response.Body(), nil
}
func (httpClient *_HttpClient) DoDeadlineByContext(cxt ClientContext, deadline time.Time) (int, []byte, error) {
	cxt.InjectRequestHeader()
	response := cxt.GetResponse()
	err := httpClient.DoDeadline(cxt.GetRequest(), response, deadline)
	if err != nil {
		return 0, nil, err
	}
	cxt.HandleResponseHeader()
	defer cxt.Release()
	return response.StatusCode(), response.Body(), nil
}
func (httpClient *_HttpClient) DoByContext(cxt ClientContext) (int, []byte, error) {
	cxt.InjectRequestHeader()
	response := cxt.GetResponse()
	err := httpClient.Do(cxt.GetRequest(), cxt.GetResponse())
	if err != nil {
		return 0, nil, err
	}
	cxt.HandleResponseHeader()
	defer cxt.Release()
	return response.StatusCode(), response.Body(), nil
}
