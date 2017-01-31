package httpclient

import (
	"sync"

	"strings"

	"bytes"

	"github.com/coffeehc/commons/convers"
)

//NewCookieManager 创建一个新的 CookieManager
func NewCookieManager() CookieManager {
	return &_CookieManager{
		cookies: make(map[string][]Cookie),
		mutex:   new(sync.Mutex),
	}
}

type _CookieManager struct {
	cookies map[string][]Cookie
	mutex   *sync.Mutex
}

func (m *_CookieManager) SetCookie(cookie Cookie) {
	d := cookie.Domain()
	if len(d) == 0 {
		return
	}
	if d[0] == '*' {
		d = d[1:]
	}
	domain := convers.BytesToString(d)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	cookies, ok := m.cookies[domain]
	if !ok {
		cookies = make([]Cookie, 0)
	}
	for i, c := range cookies {
		if bytes.Equal(c.Key(), cookie.Key()) {
			cookies[i] = cookie
			return
		}
	}
	cookies = append(cookies, cookie)
	m.cookies[domain] = cookies
}
func (m *_CookieManager) GetCookies(domain string) []Cookie {
	cs := make([]Cookie, 0)
	for d, cookies := range m.cookies {
		if strings.HasSuffix(domain, d) {
			cs = append(cs, cookies...)
		}
	}
	return cs
}
