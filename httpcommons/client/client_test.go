package client_test

import (
	"testing"

	"github.com/coffeehc/commons/httpcommons/client"
	"github.com/coffeehc/logger"
)

func Test_Client_Do(t *testing.T) {
	logger.SetDefaultLevel("/", logger.LevelDebug)
	_client := client.NewHTTPClient(nil)
	resp, err := _client.Get("http://www.163.com")
	if err != nil {
		t.Fatalf("error is %s", err)
		t.FailNow()
	}
	body, err := client.ReadBody(resp, "")
	if err != nil {
		t.Fatalf("error is %s", err)
		t.FailNow()
	}
	t.Logf("body is %s", body)
	request := client.NewRequest()
	t.Logf("request is %v", request)
	request.SetURI("http://www.baidu.com")
	resp, err = _client.Do(request)
	if err != nil {
		t.Fatalf("error is %s", err)
		t.FailNow()
	}
	body, err = client.ReadBody(resp, "")
	if err != nil {
		t.Fatalf("error is %s", err)
		t.FailNow()
	}
	t.Logf("body is %s", body)

}
