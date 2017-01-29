package httpclient

import "github.com/valyala/fasthttp"

var defaultHTTPClientOption = &Option{}

//NewHTTPClient 创建一个新的 HttpClient
func NewHTTPClient(clientOption *Option) Client {
	if clientOption == nil {
		clientOption = defaultHTTPClientOption
	}
	client := &fasthttp.Client{
		Name:                          clientOption.GetName(),
		Dial:                          clientOption.GetDial(),
		DialDualStack:                 clientOption.GetDialDualStack(),
		TLSConfig:                     clientOption.GetTLSConfig(),
		MaxConnsPerHost:               clientOption.GetMaxConnsPerHost(),
		MaxIdleConnDuration:           clientOption.GetMaxIdleConnDuration(),
		ReadBufferSize:                clientOption.GetReadBufferSize(),
		WriteBufferSize:               clientOption.GetWriteBufferSize(),
		ReadTimeout:                   clientOption.GetReadTimeout(),
		WriteTimeout:                  clientOption.GetWriteTimeout(),
		MaxResponseBodySize:           clientOption.GetMaxResponseBodySize(),
		DisableHeaderNamesNormalizing: true,
	}
	return &_HttpClient{
		client: client,
	}
}

//NewHTTPArgs 创建一个新的参数
func NewHTTPArgs() Args {
	return &_Args{
		args: fasthttp.AcquireArgs(),
	}
}

// ReleaseArgs 释放 Args对象
func ReleaseArgs(args Args) {
	fasthttp.ReleaseArgs(args.getFastHTTPArgs())
}

//NewHTTPCookie 创建一个 Cookie
func NewHTTPCookie() Cookie {
	return &_Cookie{
		cookie: fasthttp.AcquireCookie(),
	}
}

//ReleaseCookie 释放 Cookie 对象
func ReleaseCookie(cookie Cookie) {
	fasthttp.ReleaseCookie(cookie.getFastHTTPCookie())
}

//NewHTTPURI 创建一个新HTTPUri
func NewHTTPURI() URI {
	return &_HttpURI{
		uri: fasthttp.AcquireURI(),
	}
}

//ReleaseURI 释放 URI 对象
func ReleaseURI(uri URI) {
	fasthttp.ReleaseURI(uri.getFastHTTPURI())
}

//NewHTTPRequest 创建新的 Request
func NewHTTPRequest() Request {
	return &_Request{
		request: fasthttp.AcquireRequest(),
	}
}

//ReleaseRequest 释放 Request 对象
func ReleaseRequest(request Request) {
	fasthttp.ReleaseRequest(request.getFastHTTPRequest())
}

//NewHTTPResponse 创建新的 Response
func NewHTTPResponse() Response {
	return &_Response{
		response: fasthttp.AcquireResponse(),
	}
}

//ReleaseResponse 释放Response对象
func ReleaseResponse(request Response) {
	fasthttp.ReleaseResponse(request.getFastHTTPResponse())
}
