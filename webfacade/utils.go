package webfacade

import (
	"encoding/json"
	"fmt"
	"github.com/coffeehc/commons/coder"
	"github.com/coffeehc/commons/sequences"
	"github.com/coffeehc/commons/utils"
	"github.com/gofiber/fiber/v2"
	"google.golang.org/protobuf/proto"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/coffeehc/base/errors"
	"github.com/coffeehc/base/log"
	"go.uber.org/zap"
)

func SendPBSuccess(c *fiber.Ctx, data proto.Message, code int64) error {
	resp := &PBResponse{
		Success: true,
		Code:    code,
	}
	payload, e := coder.PBCoder.Marshal(data)
	if e != nil {
		resp.Success = false
		resp.Message = "返回数据序列化失败"
	} else {
		resp.Payload = payload
	}
	body, err := coder.PBCoder.Marshal(resp)
	if err != nil {
		return c.SendStatus(500)
	}
	return c.Send(body)
}

func SendPBErrors(c *fiber.Ctx, err error, code int64, statusCode int) error {
	message := err.Error()
	if errors.IsSystemError(err) {
		log.Error("遭遇了系统错误", zap.Error(err))
		message = "系统忙，请稍后再试"
	}
	return SendPBError(c, message, code, statusCode)
}

func SendPBError(c *fiber.Ctx, message string, code int64, statusCode int) error {
	resp := &PBResponse{
		Success: false,
		Code:    code,
		Message: message,
	}
	body, err := coder.PBCoder.Marshal(resp)
	if err != nil {
		return c.SendStatus(500)
	}
	return c.Send(body)
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

func GetRemortIp(c *fiber.Ctx) string {
	ip := c.Get("X-Forwarded-For")
	if ip != "" {
		return strings.Split(ip, ",")[0]
	}
	ip = c.Get("X-Real-Ip")
	if ip != "" {
		return strings.Split(ip, ",")[0]
	}
	return c.IP()
}

func WriteContentType(w http.ResponseWriter, value []string) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = value
	}
}

func ReadIds(ids []string) []int64 {
	_ids := make([]int64, 0, len(ids))
	for _, id := range ids {
		_id, _ := strconv.ParseInt(id, 10, 64)
		if _id != 0 {
			_ids = append(_ids, _id)
		}
	}
	return _ids
}
