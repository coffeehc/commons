package middlewares

import (
	"encoding/base64"
	"github.com/coffeehc/commons/utils"
	"github.com/coffeehc/commons/webfacade"
	"net/http"
	"sync"

	"github.com/coffeehc/base/errors"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

const TokenAuthUserKey = "_tokenAuthUserKey"

type AccountAuthType int64

const (
	AccountAuthType_BaseAuth AccountAuthType = 1
	AccountAuthType_Token    AccountAuthType = 2 // 兼容jwt
)

type Account struct {
	Id       int64
	Username string
	Token    string
	Rate     float64
	Bursts   int
}

type TokenAuthService interface {
	GetTokenKey() string
	CheckQuery() bool
	Auth(token string) *Account
	AddAccount(account *Account) error
	DelAccount(account *Account)
	GetLimiter(username string) *rate.Limiter
}

func TokenAuthMiddleware(accountAuthService TokenAuthService) gin.HandlerFunc {
	tokenKey := accountAuthService.GetTokenKey()
	checkQuery := accountAuthService.CheckQuery()
	return func(c *gin.Context) {
		token := c.GetHeader(tokenKey)
		if token == "" && checkQuery {
			token = c.Query("token")
		}
		if token != "" {
			account := accountAuthService.Auth(token)
			if account != nil {
				limiter := accountAuthService.GetLimiter(account.Username)
				if !limiter.Allow() {
					resp := &webfacade.AjaxResponse{
						Message:   "请求频繁",
						Code:      http.StatusTooManyRequests,
						RequestID: utils.Int64IdEncode(GetRequestId(c)),
					}
					c.Render(http.StatusTooManyRequests, &webfacade.JsonRender{
						Data: resp,
					})
					c.Abort()
					return
				}
				c.Set(TokenAuthUserKey, account)
				c.Next()
				return
			}
		}
		resp := &webfacade.AjaxResponse{
			Message:   "认证失败",
			Code:      http.StatusUnauthorized,
			RequestID: utils.Int64IdEncode(GetRequestId(c)),
		}
		c.Render(http.StatusUnauthorized, &webfacade.JsonRender{
			Data: resp,
		})
		c.Abort()
	}
}

func NewTokenAuthService(tokenKey string, checkQuery bool, defaultRate float64, defaultBursts int) TokenAuthService {
	impl := &authServiceImpl{
		tokenKey:       tokenKey,   // "Authorization",
		checkQuery:     checkQuery, // false,
		defaultLimiter: rate.NewLimiter(rate.Limit(defaultRate), defaultBursts),
	}
	return impl
}

type authServiceImpl struct {
	tokenKey       string
	checkQuery     bool
	tokens         sync.Map
	limiters       sync.Map
	defaultLimiter *rate.Limiter
}

func (impl *authServiceImpl) GetTokenKey() string {
	return impl.tokenKey
}
func (impl *authServiceImpl) CheckQuery() bool {
	return impl.checkQuery
}
func (impl *authServiceImpl) Auth(token string) *Account {
	v, ok := impl.tokens.Load(token)
	if ok {
		return v.(*Account)
	}
	return nil
}

func (impl *authServiceImpl) GetLimiter(username string) *rate.Limiter {
	v, ok := impl.limiters.Load(username)
	if ok {
		return v.(*rate.Limiter)
	}
	return impl.defaultLimiter
}

func (impl *authServiceImpl) AddAccount(account *Account) error {
	_, ok := impl.tokens.LoadOrStore(account.Token, account)
	if ok {
		return errors.MessageError("该Token已经存在")
	}
	return nil
}

func (impl *authServiceImpl) DelAccount(account *Account) {
	impl.tokens.Delete(account.Token)
}

func (impl *authServiceImpl) addLimiter(account *Account) {
	if account.Rate != 0 {
		limiter := rate.NewLimiter(rate.Limit(account.Rate), account.Bursts)
		impl.limiters.Store(account.Username, limiter)
	}
}

func (impl *authServiceImpl) delLimiter(account *Account) {
	impl.limiters.Delete(account.Username)
}

func NewBaseTokenAuthService(defaultRate float64, defaultBursts int) TokenAuthService {
	impl := &baseAuthServiceImpl{&authServiceImpl{
		tokenKey:       "Authorization",
		checkQuery:     false,
		defaultLimiter: rate.NewLimiter(rate.Limit(defaultRate), defaultBursts),
	}}
	return impl
}

type baseAuthServiceImpl struct {
	*authServiceImpl
}

func (impl *baseAuthServiceImpl) AddAccount(account *Account) error {
	token := "Basic " + base64.StdEncoding.EncodeToString([]byte(account.Username+":"+account.Token))
	_, ok := impl.tokens.LoadOrStore(token, account)
	if ok {
		return errors.MessageError("该Token已经存在")
	}
	impl.addLimiter(account)
	return nil
}

func (impl *baseAuthServiceImpl) DelAccount(account *Account) {
	token := "Basic " + base64.StdEncoding.EncodeToString([]byte(account.Username+":"+account.Token))
	impl.tokens.Delete(token)
	impl.delLimiter(account)
}
