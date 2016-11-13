package httpclient


func NewHttpClientService(defaultName string) *HttpClientService{
	if defaultName == ""{
		defaultName = "coffee's go http client"
	}
	return &HttpClientService{
		defaultName:defaultName,
	}
}

type HttpClientService struct {
	defaultName  string
	client HttpClient
}

func (this *HttpClientService)getClient() HttpClient{
	if this.client == nil{
		option := &HttpClientOption{
			Name:this.defaultName,
		}
		this.client = this.newClient(option)
	}
	return  this.client
}

func (this *HttpClientService)newClient(option *HttpClientOption)HttpClient{
	return NewHttpClient(option)
}
