package client

import (
	"io"
	"io/ioutil"
	"net/http"
)

func NewHTTPClient(defaultOptions *HTTPClientOptions) HTTPClient {
	if defaultOptions == nil {
		defaultOptions = &HTTPClientOptions{}
	}
	return &_Client{
		defaultOptions: defaultOptions,
	}
}

type _Client struct {
	defaultOptions *HTTPClientOptions
}

func (c *_Client) Get(url string) (HTTPResponse, error) {
	req := NewRequest()
	req.SeMethod("GET")
	req.SetURI(url)
	return c.Do(req)
}

func (c *_Client) POST(url string, body io.Reader, contentType string) (HTTPResponse, error) {
	req := NewRequest()
	req.SeMethod("POST")
	req.SetURI(url)
	var readerCloser io.ReadCloser
	if rc, ok := body.(io.ReadCloser); ok {
		readerCloser = rc
	} else {
		readerCloser = ioutil.NopCloser(body)
	}
	req.SetBodyStream(readerCloser)
	req.SetContentType(contentType)
	return c.Do(req)
}

func (c *_Client) Do(req HTTPRequest) (HTTPResponse, error) {
	_req := req.(*_HTTPRequest)
	client := new(http.Client)
	c.defaultOptions.setClientOptions(client)
	//TODO 组装 Request
	if _req.cookieJar != nil {
		client.Jar = _req.cookieJar
	}
	resp, err := client.Do(_req.GetRealRequest())
	if err != nil {
		return nil, err
	}
	return newResponse(resp), nil
}
