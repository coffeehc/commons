package webfacade

import (
	"encoding/json"
	"fmt"
	"github.com/coffeehc/commons/coder"
	"github.com/coffeehc/commons/sequences"
	"github.com/coffeehc/commons/utils"
	"github.com/coffeehc/commons/webfacade/internal"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/coffeehc/base/errors"
	"github.com/coffeehc/base/log"
	"github.com/gin-gonic/gin"
	"github.com/gogo/protobuf/proto"
	"go.uber.org/zap"
)

func SendPBSuccess(c *gin.Context, data proto.Message, code int64) {
	resp := &PBResponse{
		RequestId: GetRequestId(c),
		Success:   true,
		Code:      code,
	}
	payload, e := coder.PBCoder.Marshal(data)
	if e != nil {
		resp.Success = false
		resp.Message = "返回数据序列化失败"
	} else {
		resp.Payload = payload
	}
	c.Render(http.StatusOK, &ProtobufRender{
		Data: resp,
	})
	c.Abort()
}

func SendPBErrors(c *gin.Context, err error, code int64, statusCode int) {
	message := err.Error()
	if errors.IsSystemError(err) {
		log.Error("遭遇了系统错误", zap.Error(err))
		message = "系统忙，请稍后再试"
	}
	SendPBError(c, message, code, statusCode)
}

func SendPBError(c *gin.Context, message string, code int64, statusCode int) {
	resp := &PBResponse{
		RequestId: GetRequestId(c),
		Success:   false,
		Code:      code,
		Message:   message,
	}
	c.Render(statusCode, &ProtobufRender{
		Data: resp,
	})
	c.Abort()
}

func GetRequestId(c *gin.Context) int64 {
	return c.GetInt64(internal.ContextKey_RequestId)
}

func FormatSequence(layout string, id int64) string {
	if id == 0 {
		return ""
	}
	createTime := sequences.ParseSequenceToTime(id)
	return fmt.Sprintf("%s%s", createTime.Format(layout), utils.Int64IdEncode(id))
}

func ReadPBResponse(body []byte, payload proto.Message) error {
	response := &PBResponse{}
	e := coder.PBCoder.Unmarshal(body, response)
	if e != nil {
		log.Error("解析失败", zap.String("body", string(body)), zap.Error(e))
		return errors.ConverError(e)
	}
	if !response.Success {
		return errors.BuildError(response.GetCode(), response.GetMessage())
	}
	e = coder.PBCoder.Unmarshal(response.GetPayload(), payload)
	if e != nil {
		log.Error("解析失败", zap.String("body", string(body)), zap.Error(e))
	}
	return errors.ConverError(e)
}

func ReaderBodyByJsonFromBody(body io.ReadCloser, t interface{}) {
	defer body.Close()
	data, err := ioutil.ReadAll(body)
	if err != nil {

		panic(errors.MessageError("请求数据错误"))
	}
	err = json.Unmarshal(data, t)
	if err != nil {
		panic(errors.MessageError("无法解析请求数据"))
	}
}

func GetRemortIp(c *gin.Context) string {
	ip := c.GetHeader("X-Forwarded-For")
	if ip != "" {
		return strings.Split(ip, ",")[0]
	}
	ip = c.GetHeader("X-Real-Ip")
	if ip != "" {
		return strings.Split(ip, ",")[0]
	}
	return c.ClientIP()
}

func WriteContentType(w http.ResponseWriter, value []string) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = value
	}
}
