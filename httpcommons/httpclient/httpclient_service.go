package httpclient

//NewHTTPClientService 创建新的 HttpClientServie
func NewHTTPClientService(defaultName string) *Service {
	if defaultName == "" {
		defaultName = "coffee's go http client"
	}
	return &Service{
		defaultName: defaultName,
	}
}

//Service  a http client Service
type Service struct {
	defaultName string
	client      Client
}

//GetGlobalClient 获取全局的使用的 Client
func (service *Service) GetGlobalClient() Client {
	if service.client == nil {
		option := &Option{
			Name: service.defaultName,
		}
		service.client = service.NewClient(option)
	}
	return service.client
}

//NewClient 创建一个新的 Client
func (*Service) NewClient(option *Option) Client {
	return NewHTTPClient(option)
}
