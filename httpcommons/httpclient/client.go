package httpclient

import (
	"time"
	"github.com/valyala/fasthttp"
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
	return this.client.Post(dst,url,postArgs.getFastHttpAtgs())
}
func (this *_HttpClient)DoTimeout(req HttpRequest, resp HttpResponse, timeout time.Duration) error{
	return this.client.DoTimeout(req.getFastHttpRequest(),resp.getFastHttpResponse(),timeout)
}
func (this *_HttpClient)DoDeadline(req HttpRequest, resp HttpResponse, deadline time.Time) error{
	return this.client.DoDeadline(req.getFastHttpRequest(),resp.getFastHttpResponse(),deadline)
}
func (this *_HttpClient)Do(req HttpRequest, resp HttpResponse) error{
	return this.client.Do(req.getFastHttpRequest(),resp.getFastHttpResponse())
}
