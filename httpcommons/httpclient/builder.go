package httpclient

import "github.com/valyala/fasthttp"

func NewHttpClient(clientOption *HttpClientOption) HttpClient {
	client := &fasthttp.Client{
		Name: clientOption.GetName(),
		Dial: clientOption.GetDial(),
		DialDualStack:clientOption.GetDialDualStack(),
		TLSConfig:clientOption.GetTLSConfig(),
		MaxConnsPerHost:clientOption.GetMaxConnsPerHost(),
		MaxIdleConnDuration:clientOption.GetMaxIdleConnDuration(),
		ReadBufferSize:clientOption.GetReadBufferSize(),
		WriteBufferSize:clientOption.GetWriteBufferSize(),
		ReadTimeout:clientOption.GetReadTimeout(),
		WriteTimeout:clientOption.GetWriteTimeout(),
		MaxResponseBodySize:clientOption.GetMaxResponseBodySize(),
		DisableHeaderNamesNormalizing:true,
	}
	return &_HttpClient{
		client: client,
	}
	return nil
}

func NewHttpArgs() HttpArgs  {
	return &_HttpArgs{
		args:fasthttp.AcquireArgs(),
	}
}

func NewHttpCookie() HttpCookie{
	return &_HttpCookie{
		cookie:fasthttp.AcquireCookie(),
	}
}

func NewHttpURI() HttpUri {
	return &_HttpUri{
		uri:fasthttp.AcquireURI(),
	}
	
}

func NewHttpRequest() HttpRequest{
	return &_HttpRequest{
		request:fasthttp.AcquireRequest(),
	}
}

func NreHttpResponse() HttpResponse{
	return &_HttpResponse{
		response:fasthttp.AcquireResponse(),
	}
}
