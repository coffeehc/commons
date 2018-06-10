package client_test

import (
	"bytes"
	"testing"
	"time"

	"git.xiagaogao.com/coffee/commons/convers"
	"git.xiagaogao.com/coffee/commons/https/client"
	"go.uber.org/zap"
)

func Test_Client_Do(t *testing.T) {
	logger,_ := zap.NewDevelopment()
	option := &client.HTTPClientOptions{
		Timeout:       3 * time.Second,
		DialerTimeout: 3 * time.Second,
	}
	option.AddHeaderSetting(client.NewHeaderUserAgent("123"))
	_client := client.NewHTTPClient(option, option.NewTransport(nil))
	dataStr := `ip=myip`
	resp, err := _client.POST("http://ip.taobao.com/service/getIpInfo2.php", bytes.NewReader(convers.StringToBytes(dataStr)), "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatalf("error is %#v[%s]", err, err.Error())
		t.FailNow()
	}
	body, err := client.ReadBody(resp, "")
	t.Logf("body is %s", body)
	if err != nil {
		t.Fatalf("error is %#v[%s]", err, err.Error())
		t.FailNow()
	}
	proxyDialer, err := client.NewProxyDialer("socks5://127.0.0.1:1080", option.NewDialer())
	if err != nil {
		t.Fatalf("error is %#v[%s]", err, err.Error())
		t.FailNow()
	}
	transport := option.NewTransport(proxyDialer.DialContext)
	//transport.Proxy, _ = client.NewProxyByAddrProviter(&client.AddrsProxyProvicer{
	//	HttpProxys:  []string{"110.164.58.147:9001"},
	//	HttpsProxys: []string{},
	//})
	req, _ := client.NewHTTPRequest("POST", "http://ip.taobao.com/service/getIpInfo2.php")
	req.SetTransport(transport)
	req.SetBody(convers.StringToBytes(dataStr))
	req.SetContentType("application/x-www-form-urlencoded")
	resp, err = _client.Do(req, false)
	if err != nil {
		t.Fatalf("error is %#v[%s]", err, err.Error())
		t.FailNow()
	}
	body, err = client.ReadBody(resp, "")
	if err != nil {
		t.Fatalf("error is %s", err)
		t.FailNow()
	}
	t.Logf("body is %s", body)
	time.Sleep(time.Millisecond * 300)
}
