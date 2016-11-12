package httpclient

import (
	"time"
	"github.com/valyala/fasthttp"
	"errors"
)

type _HttpClient struct {
	client *fasthttp.Client
}

func (this *_HttpClient)Get(dst []byte, url string) (statusCode int, body []byte, err error) {
	return this.client.Get(dst,url)
}
func (this *_HttpClient)GetTimeout(dst []byte, url string, timeout time.Duration) (statusCode int, body []byte, err error){
	return this.client.GetTimeout(dst,url,timeout)
}
func (this *_HttpClient)GetDeadline(dst []byte, url string, deadline time.Time) (statusCode int, body []byte, err error){
	return this.client.GetDeadline(dst,url,deadline)
}
func (this *_HttpClient)Post(dst []byte, url string, postArgs HttpArgs) (statusCode int, body []byte, err error){
	var args *fasthttp.Args
	if postArgs != nil {
		args = postArgs.getFastHttpAtgs()
	}
	return this.client.Post(dst, url,args)
}
func (this *_HttpClient)DoTimeout(req HttpRequest, resp HttpResponse, timeout time.Duration) error{
	if req == nil{
		return errors.New("Request 不能为 nil")
	}
	if resp == nil{
		return errors.New("Request 不能为 nil")
	}
	return this.client.DoTimeout(req.getFastHttpRequest(),resp.getFastHttpResponse(),timeout)
}
func (this *_HttpClient)DoDeadline(req HttpRequest, resp HttpResponse, deadline time.Time) error{
	if req == nil{
		return errors.New("Request 不能为 nil")
	}
	if resp == nil{
		return errors.New("Request 不能为 nil")
	}
	return this.client.DoDeadline(req.getFastHttpRequest(),resp.getFastHttpResponse(),deadline)
}
func (this *_HttpClient)Do(req HttpRequest, resp HttpResponse) error{
	if req == nil{
		return errors.New("Request 不能为 nil")
	}
	if resp == nil{
		return errors.New("Request 不能为 nil")
	}
	return this.client.Do(req.getFastHttpRequest(),resp.getFastHttpResponse())
}
