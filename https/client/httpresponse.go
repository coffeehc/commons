package client

import (
	"io"
	"net/http"

	"git.xiagaogao.com/coffee/boot/errors"
	"go.uber.org/zap"
)

func newHTTPResponse(resp *http.Response,errorService errors.Service,logger *zap.Logger) HTTPResponse {
	return &httpResponseImpl{
		resp: resp,
		errorService:errorService,
		logger:logger,
	}
}

type httpResponseImpl struct {
	resp *http.Response
	errorService errors.Service
	logger *zap.Logger
}

func ( impl *httpResponseImpl) GetBody() io.ReadCloser {
	return impl.resp.Body
}
func ( impl *httpResponseImpl) GetRealResponse() *http.Response {
	return impl.resp
}

func ( impl *httpResponseImpl) GetHeader() http.Header {
	return impl.resp.Header
}

func ( impl *httpResponseImpl) GetContentType() string {
	return impl.resp.Header.Get("Content-Type")
}

func ( impl *httpResponseImpl) GetStatusCode() int {
	return impl.resp.StatusCode
}

func ( impl *httpResponseImpl) GetCookies() []*http.Cookie {
	return impl.resp.Cookies()
}
