package client

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"net/http/cookiejar"

	"git.xiagaogao.com/coffee/boot/errors"
	"go.uber.org/zap"
)

func NewHTTPClient(defaultOptions *HTTPClientOptions, globalTransport *http.Transport,errorService errors.Service,logger *zap.Logger) HTTPClient {
	errorService = errorService.NewService("httpclient")
	if defaultOptions == nil {
		defaultOptions = &HTTPClientOptions{}
	}
	if globalTransport == nil {
		globalTransport = defaultOptions.NewTransport(nil)
	}
	return &clientImpl{
		options:          defaultOptions,
		defaultTransport: globalTransport,
		timeout:          defaultOptions.GetTimeout(),
		errorService:errorService,
		logger:logger,
	}
}

type clientImpl struct {
	options          *HTTPClientOptions
	defaultTransport *http.Transport
	timeout          time.Duration
	errorService errors.Service
	logger *zap.Logger
}

func (impl *clientImpl) Config() *HTTPClientOptions {
	return impl.options
}

func (impl *clientImpl) Get(url string) (HTTPResponse, error) {
	req, err := NewHTTPRequest("GET", url,impl.errorService,impl.logger)
	if err != nil {
		return nil, err
	}
	req.SetMethod("GET")
	req.SetURI(url)
	return impl.Do(req, true)
}

func (impl *clientImpl) POST(url string, body io.Reader, contentType string) (HTTPResponse, error) {
	req, err := NewHTTPRequest("POST", url,impl.errorService,impl.logger)
	if err != nil {
		return nil, err
	}
	req.SetURI(url)
	var readerCloser io.ReadCloser
	if rc, ok := body.(io.ReadCloser); ok {
		readerCloser = rc
	} else {
		readerCloser = ioutil.NopCloser(body)
	}
	req.SetBodyStream(readerCloser)
	req.SetContentType(contentType)
	return impl.Do(req, true)
}

func (impl *clientImpl) Do(req HTTPRequest, autoRedirect bool) (HTTPResponse, error) {
	_req := impl.init(req)
	resp, err := impl.do(_req, autoRedirect)
	if err != nil {
		return nil, err
	}
	//TODO 异步关闭response的body,防止使用者没有关闭body
	go func() {
		timeout := impl.Config().Timeout
		if timeout == 0 || timeout > time.Second*5 {
			timeout = time.Second * 3
		}
		time.Sleep(timeout)
		if !resp.Close && resp.Body != nil {
			resp.Body.Close()
		}
		req := _req.GetRealRequest()
		if !req.Close && req.Body != nil {
			req.Body.Close()
		}
	}()
	return newHTTPResponse(resp,impl.errorService,impl.logger), nil
}

func (impl *clientImpl) do(req *httpRequestImpl, autoRedirect bool) (*http.Response, error) {
	realRequest := req.GetRealRequest()
	impl.options.setHeader(realRequest)
	if autoRedirect {
		method := realRequest.Method
		if method == "GET" || method == "HEAD" {
			return impl.doFollowingRedirects(impl.timeout, req, impl.shouldRedirectGet)
		}
		if method == "POST" || method == "PUT" {
			return impl.doFollowingRedirects(impl.timeout, req, impl.shouldRedirectPost)
		}
	}
	return impl.send(req)
}

func (impl *clientImpl) init(req HTTPRequest) *httpRequestImpl {
	_req := req.(*httpRequestImpl)
	if _req.transport == nil {
		_req.transport = impl.defaultTransport
	}
	realRequest := req.GetRealRequest()
	if _req.cookieJar != nil {
		for _, cookie := range _req.cookieJar.Cookies(realRequest.URL) {
			realRequest.AddCookie(cookie)
		}
	}
	//TODO 是否要处理Cookie
	return _req
}

func (impl *clientImpl) send(req *httpRequestImpl) (*http.Response, error) {
	deadline := deadline(impl.timeout)
	resp, err := send(req, deadline)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func send(_req *httpRequestImpl, deadline time.Time) (*http.Response, error) {
	//ireq *http.Request, rt http.RoundTripper
	ireq := _req.GetRealRequest()
	rt := _req.transport
	req := ireq // req is either the original request, or a modified fork

	if rt == nil {
		closeBody(req.Body)
		return nil, _req.errorService.SystemError("http: no Client.Transport or DefaultTransport")
	}

	if req.URL == nil {
		closeBody(req.Body)
		return nil, _req.errorService.SystemError("http: nil Request.URL")
	}

	if req.RequestURI != "" {
		closeBody(req.Body)
		return nil, _req.errorService.SystemError("http: Request.RequestURI can't be set in client requests.")
	}

	// forkReq forks req into a shallow clone of ireq the first
	// time it's called.
	forkReq := func() {
		if ireq == req {
			req = new(http.Request)
			*req = *ireq // shallow clone
		}
	}

	// Most the callers of send (Get, Post, et al) don't need
	// Headers, leaving it uninitialized. We guarantee to the
	// Transport that this has been initialized, though.
	if req.Header == nil {
		forkReq()
		req.Header = make(http.Header)
	}

	if u := req.URL.User; u != nil && req.Header.Get("Authorization") == "" {
		username := u.Username()
		password, _ := u.Password()
		forkReq()
		req.Header = cloneHeader(ireq.Header)
		req.Header.Set("Authorization", "Basic "+basicAuth(username, password))
	}

	if !deadline.IsZero() {
		forkReq()
	}
	stopTimer, wasCanceled := setRequestCancel(req, rt, deadline)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		stopTimer()
		if resp != nil {
			_req.logger.Warn("RoundTripper returned a response & error; ignoring response")
		}
		if tlsErr, ok := err.(tls.RecordHeaderError); ok {
			// If we get a bad TLS record header, check to see if the
			// response looks like HTTP and give a more helpful error.
			// See golang.org/issue/11111.
			if string(tlsErr.RecordHeader[:]) == "HTTP/" {
				err = _req.errorService.SystemError("http: server gave HTTP response to HTTPS client")
			}
		}
		return nil, err
	}
	//设置Cookie
	if _req.cookieJar != nil {
		_url := req.URL
		if rc := resp.Cookies(); len(rc) > 0 {
			_req.cookieJar.SetCookies(_url, rc)
		}
	}
	if !deadline.IsZero() {
		resp.Body = &cancelTimerBody{
			stop:           stopTimer,
			rc:             resp.Body,
			reqWasCanceled: wasCanceled,
		}
	}
	return resp, nil
}

func (impl *clientImpl)doFollowingRedirects(timeout time.Duration, _req *httpRequestImpl, shouldRedirect func(int) bool) (*http.Response, error) {
	req := _req.GetRealRequest()
	if req.URL == nil {
		closeBody(req.Body)
		return nil, _req.errorService.SystemError("http: nil Request.URL")
	}

	var (
		deadline = deadline(timeout)
		reqs     []*http.Request
		resp     *http.Response
	)
	uerr := func(err error) error {
		closeBody(req.Body)
		method := valueOrDefault(reqs[0].Method, "GET")
		var urlStr string
		if resp != nil && resp.Request != nil {
			urlStr = resp.Request.URL.String()
		} else {
			urlStr = req.URL.String()
		}
		return &url.Error{
			Op:  method[:1] + strings.ToLower(method[1:]),
			URL: urlStr,
			Err: err,
		}
	}
	for {
		// For all but the first request, create the next
		// request hop and replace req.
		if len(reqs) > 0 {
			loc := resp.Header.Get("Location")
			if loc == "" {
				return nil, uerr(fmt.Errorf("%d response missing Location header", resp.StatusCode))
			}
			u, err := req.URL.Parse(loc)
			if err != nil {
				return nil, uerr(fmt.Errorf("failed to parse Location header %q: %v", loc, err))
			}
			ireq := reqs[0]
			req = &http.Request{
				Method:   ireq.Method,
				Response: resp,
				URL:      u,
				Header:   make(http.Header),
				Cancel:   ireq.Cancel,
				//ctx:      ireq.ctx,
			}
			req.WithContext(ireq.Context())
			if ireq.Method == "POST" || ireq.Method == "PUT" {
				req.Method = "GET"
			}
			// Add the Referer header from the most recent
			// request URL to the new one, if it's not https->http:
			if ref := refererForURL(reqs[len(reqs)-1].URL, req.URL); ref != "" {
				req.Header.Set("Referer", ref)
			}
			err = impl.defaultCheckRedirect(req, reqs)
			// Sentinel error to let users select the
			// previous response, without closing its
			// body. See Issue 10069.
			if err == http.ErrUseLastResponse {
				return resp, nil
			}

			// Close the previous response's body. But
			// read at least some of the body so if it's
			// small the underlying TCP connection will be
			// re-used. No need to check for errors: if it
			// fails, the Transport won't reuse it anyway.
			const maxBodySlurpSize = 2 << 10
			if resp.ContentLength == -1 || resp.ContentLength <= maxBodySlurpSize {
				io.CopyN(ioutil.Discard, resp.Body, maxBodySlurpSize)
			}
			resp.Body.Close()

			if err != nil {
				// Special case for Go 1 compatibility: return both the response
				// and an error if the CheckRedirect function failed.
				// See https://golang.org/issue/3795
				// The resp.Body has already been closed.
				ue := uerr(err)
				ue.(*url.Error).URL = loc
				return resp, ue
			}
			_req.req = req
		}

		reqs = append(reqs, req)

		var err error
		if resp, err = send(_req, deadline); err != nil {
			if !deadline.IsZero() && !time.Now().Before(deadline) {
				err = &httpError{
					err:     err.Error() + " (Client.Timeout exceeded while awaiting headers)",
					timeout: true,
				}
			}
			return nil, uerr(err)
		}

		if !shouldRedirect(resp.StatusCode) {
			return resp, nil
		}
		if len(resp.Cookies()) > 0 {
			if _req.cookieJar == nil {
				_req.cookieJar, _ = cookiejar.New(nil)
			}
			_req.cookieJar.SetCookies(resp.Request.URL, resp.Cookies())
		}
	}
}

func refererForURL(lastReq, newReq *url.URL) string {
	// https://tools.ietf.org/html/rfc7231#section-5.5.2
	//   "Clients SHOULD NOT include a Referer header field in a
	//    (non-secure) HTTP request if the referring page was
	//    transferred with a secure protocol."
	if lastReq.Scheme == "https" && newReq.Scheme == "http" {
		return ""
	}
	referer := lastReq.String()
	if lastReq.User != nil {
		// This is not very efficient, but is the best we can
		// do without:
		// - introducing a new method on URL
		// - creating a race condition
		// - copying the URL struct manually, which would cause
		//   maintenance problems down the line
		auth := lastReq.User.String() + "@"
		referer = strings.Replace(referer, auth, "", 1)
	}
	return referer
}

//TODO 这个使用默认的,暂时是不需要单独处理的
func (impl *clientImpl)defaultCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return impl.errorService.SystemError("stopped after 10 redirects")
	}
	return nil
}

// True if the specified HTTP status code is one for which the Get utility should
// automatically redirect.
func (impl *clientImpl)shouldRedirectGet(statusCode int) bool {
	switch statusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect:
		return true
	}
	return false
}

// True if the specified HTTP status code is one for which the Post utility should
// automatically redirect.
func (impl *clientImpl)shouldRedirectPost(statusCode int) bool {
	switch statusCode {
	case http.StatusFound, http.StatusSeeOther:
		return true
	}
	return false
}
