package httpc

import (
	"net/http"
	"net/url"

	"github.com/go-resty/resty/v2"
)

func NotRedirectPolicy() resty.RedirectPolicy {
	return resty.RedirectPolicyFunc(func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	})
}

func NewNullCookieJar() http.CookieJar {
	impl := &cookieJarImpl{}
	return impl
}

type cookieJarImpl struct {
}

func (impl *cookieJarImpl) SetCookies(u *url.URL, cookies []*http.Cookie) {
}

func (impl *cookieJarImpl) Cookies(u *url.URL) []*http.Cookie {
	return nil
}
