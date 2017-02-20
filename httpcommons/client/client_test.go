package client_test

import (
	"testing"

	"bytes"
	"github.com/coffeehc/commons/convers"
	"github.com/coffeehc/commons/httpcommons/client"
	"github.com/coffeehc/logger"
	"time"
)

func Test_Client_Do(t *testing.T) {
	logger.SetDefaultLevel("/", logger.LevelDebug)
	option := &client.HTTPClientOptions{
		Timeout:       3 * time.Second,
		DialerTimeout: 3 * time.Second,
	}
	option.AddHeaderSetting(client.NewHeaderUserAgent("123"))
	_client := client.NewHTTPClient(option, option.NewTransport())
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
	transport := option.NewTransport()
	transport.Proxy, _ = client.NewProxyByAddrProviter(&client.AddrsProxyProvicer{
		HttpProxys:  []string{"110.164.58.147:9001"},
		HttpsProxys: []string{},
	})
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
