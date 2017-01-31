package httpclient

func NewClientContext(cookieManager CookieManager) ClientContext {
	cxt := &_ClientContext{}
	cxt.Reset(cookieManager)
	return cxt
}

type _ClientContext struct {
	req              Request
	resp             Response
	args             Args
	uri              URI
	cookieManager    CookieManager
	defaultReqHeader RequestHeader
	host             string
}

func (cxt *_ClientContext) Reset(cookieManager CookieManager) {
	cxt.Release()
	if cookieManager == nil {
		cookieManager = NewCookieManager()
	}
	cxt.cookieManager = cookieManager
}

func (cxt *_ClientContext) Release() {
	cxt.cookieManager = nil
	cxt.host = ""
	if cxt.req != nil {
		ReleaseRequest(cxt.req)
		cxt.req = nil
	}
	if cxt.resp != nil {
		ReleaseResponse(cxt.resp)
		cxt.resp = nil
	}
	if cxt.args != nil {
		ReleaseArgs(cxt.args)
		cxt.args = nil
	}
	if cxt.uri != nil {
		ReleaseURI(cxt.uri)
		cxt.uri = nil
	}
}

func (cxt *_ClientContext) SetDefaultRequestHeader(header RequestHeader) {
	cxt.defaultReqHeader = header
}

func (cxt *_ClientContext) GetRequest() Request {
	if cxt.req == nil {
		cxt.req = NewHTTPRequest()
		cxt.defaultReqHeader.CopyTo(cxt.req.GetRequestHeader())
	}
	return cxt.req
}
func (cxt *_ClientContext) GetResponse() Response {
	if cxt.resp == nil {
		cxt.resp = NewHTTPResponse()
	}
	return cxt.resp
}
func (cxt *_ClientContext) GetTempArgs() Args {
	if cxt.args == nil {
		cxt.args = NewHTTPArgs()
	}
	return cxt.args
}
func (cxt *_ClientContext) GetTempURI() URI {
	if cxt.uri == nil {
		cxt.uri = NewHTTPURI()
	}
	return cxt.uri
}

func (cxt *_ClientContext) InjectRequestHeader() {
	reqHeader := cxt.GetRequest().GetRequestHeader()
	cookies := cxt.cookieManager.GetCookies(cxt.GetHost())
	for _, cookie := range cookies {
		reqHeader.SetCookieBytesKV(cookie.Key(), cookie.Value())
	}
}

func (cxt *_ClientContext) HandleResponseHeader() {
	respHeader := cxt.resp.GetResponseHeader()
	defaultDomain := cxt.GetHost()
	respHeader.VisitAllCookie(func(k, v []byte) {
		cookie := NewHTTPCookie()
		e := cookie.ParseBytes(v)
		if e != nil {
			return
		}
		if len(cookie.Domain()) == 0 {
			cookie.SetDomain(defaultDomain)
		}
		cxt.cookieManager.SetCookie(cookie)
	})
}

func (cxt *_ClientContext) GetHost() string {
	if cxt.host == "" {
		uri := NewHTTPURI()
		defer ReleaseURI(uri)
		uri.UpdateBytes(cxt.req.RequestURI())
		cxt.host = string(uri.Host())
	}
	return cxt.host
}

func (cxt *_ClientContext) GetCookieManager() CookieManager {
	return cxt.cookieManager
}

func (cxt *_ClientContext) SetMethod(method string) {
	cxt.GetRequestHeader().SetMethod(method)
}
func (cxt *_ClientContext) SetMethodToGET() {
	cxt.SetMethod("GET")
}
func (cxt *_ClientContext) SetMethodToPOST() {
	cxt.SetMethod("POST")
}
func (cxt *_ClientContext) SetMethodToPUT() {
	cxt.SetMethod("PUT")
}
func (cxt *_ClientContext) SetMethodToDEL() {
	cxt.SetMethod("DELETE")
}
func (cxt *_ClientContext) SetMethodToHEAD() {
	cxt.SetMethod("HEAD")
}

func (cxt *_ClientContext) SetURI(uri string) {
	cxt.GetRequest().SetRequestURI(uri)
}

func (cxt *_ClientContext) GetRequestHeader() RequestHeader {
	return cxt.GetRequest().GetRequestHeader()
}
