package httpclient

func NewCookieFactory() CookieFactory {
	return &_CookieFactory{
		managers: make(map[string]CookieManager, 0),
	}
}

type _CookieFactory struct {
	managers map[string]CookieManager
}

func (factory *_CookieFactory) GetCookieManager(scope string) CookieManager {
	manage, ok := factory.managers[scope]
	if !ok {
		manage = NewCookieManager()
		factory.managers[scope] = manage
	}
	return manage
}
