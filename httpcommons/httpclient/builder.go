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

//NewHTTPCookie 创建一个 Cookie
func NewHTTPCookie() Cookie {
	return &_Cookie{
		cookie: fasthttp.AcquireCookie(),
	}
}

//NewHTTPURI 创建一个新HTTPUri
func NewHTTPURI() URI {
	return &_HttpURI{
		uri: fasthttp.AcquireURI(),
	}
}

//NewHTTPRequest 创建新的 Request
func NewHTTPRequest() Request {
	return &_Request{
		request: fasthttp.AcquireRequest(),
	}
}

//NewHTTPResponse 创建新的 Response
func NewHTTPResponse() Response {
	return &_Response{
		response: fasthttp.AcquireResponse(),
	}
}
