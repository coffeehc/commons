package client

import (
	"errors"
	"io"
	"io/ioutil"
	"mime"
	"net/url"
	"strings"

	"github.com/coffeehc/logger"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// ReadBodyToString 读取 body 内容
func ReadBody(resp HTTPResponse, charset string) ([]byte, error) {
	if resp == nil {
		return nil, errors.New("response is nil")
	}
	bodyReader := resp.GetBody()
	defer bodyReader.Close()
	var reader io.Reader = bodyReader
	if charset == "" {
		logger.Debug("Conetent-Type is %s", resp.GetContentType())
		_, params, _ := mime.ParseMediaType(resp.GetContentType())
		charset = params["charset"]
		charset = strings.ToUpper(charset)
		if charset == "" {
			charset = "UTF-8"
		}
	}
	//TODO 暂时支持中文解码,
	if strings.HasPrefix(charset, "GB") {
		charset = "GB13080"
		encode := simplifiedchinese.GB18030
		reader = encode.NewDecoder().Reader(bodyReader)
	}
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func BuildValues(k, v string) url.Values {
	values := make(url.Values)
	values.Set(k, v)
	return values
}

func BuildUrl(urlStr string, values url.Values) (string, error) {
	_url, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return "", err
	}
	_url.RawQuery = values.Encode()
	return _url.String(), nil
}
