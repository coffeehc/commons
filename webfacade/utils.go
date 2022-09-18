package webfacade

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"git.xiagaogao.com/base/cloudcommons/coder"
	"git.xiagaogao.com/base/cloudcommons/sequences"
	"git.xiagaogao.com/base/cloudcommons/utils"
	"git.xiagaogao.com/base/cloudcommons/webfacade/internal"
	"github.com/coffeehc/base/errors"
	"github.com/coffeehc/base/log"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
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

func AcceptHeader(c *gin.Context, contentType string) bool {
	accept := c.GetHeader("accept")
	accepts := strings.Split(accept, ",")
	for _, a := range accepts {
		if strings.Trim(strings.ToLower(a), " ") == strings.ToLower(contentType) {
			return true
		}
	}
	return false
}

func SendSuccess(c *gin.Context, json interface{}, code int64) {
	if AcceptHeader(c, "application/x-protobuf") {
		SendPBSuccess(c, json.(proto.Message), code)
		return
	}
	resp := &AjaxResponse{
		Success:   true,
		Payload:   json,
		Code:      code,
		RequestID: utils.Int64IdEncode(GetRequestId(c)),
	}
	c.Render(http.StatusOK, &JsonRender{
		Data: resp,
	})
	c.Abort()
}

func SendErrors(c *gin.Context, err error, code int64, statusCode int) {
	if AcceptHeader(c, "application/x-protobuf") {
		SendPBErrors(c, err, code, statusCode)
		return
	}
	SendErrorsWithRedirect(c, err, "", code, statusCode)
}

func SendErrorsWithRedirect(c *gin.Context, err error, redirect string, code int64, statusCode int) {
	message := err.Error()
	if errors.IsSystemError(err) {
		log.Error("遭遇了系统错误", zap.Error(err))
		message = "系统忙，请稍后再试"
	}
	SendErrorWithRedirect(c, message, redirect, code, statusCode)
}

func SendError(c *gin.Context, message string, code int64, statusCode int) {
	if AcceptHeader(c, "application/x-protobuf") {
		SendPBError(c, message, code, statusCode)
		return
	}
	SendErrorWithRedirect(c, message, "", code, statusCode)
}

func SendErrorWithRedirect(c *gin.Context, message, redirect string, code int64, statusCode int) {
	resp := &AjaxResponse{
		Message:   message,
		Code:      code,
		RequestID: utils.Int64IdEncode(GetRequestId(c)),
		Redirect:  redirect,
	}
	c.Render(statusCode, &JsonRender{
		Data: resp,
	})
	c.Abort()
}

func GetRequestId(c *gin.Context) int64 {
	return c.GetInt64(internal.ContextKey_RequestId)
}

func Bind(ginCtx *gin.Context, data interface{}) error {
	err := ginCtx.MustBindWith(data, JsonBinding)
	if err != nil {
		log.Error("无法解析请求内容", zap.Error(err))
		// panic(errors.MessageError("参数无法解析"))
		return errors.MessageError("参数无法解析")
	}
	return nil
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

func ReadResponse(resp BaseResponse, body []byte) error {
	err := json.Unmarshal(body, resp)
	if err != nil {
		return errors.ConverError(err)
	}
	if resp.IsSuccess() {
		return nil
	}
	return errors.MessageError(resp.GetMessage())
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
