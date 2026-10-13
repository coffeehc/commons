package webfacade

import (
	"strings"

	baseerrors "github.com/coffeehc/base/errors"
	"github.com/gofiber/fiber/v3"
	"google.golang.org/protobuf/proto"
)

// SendSuccess 保留既有 JSON 外壳与 protobuf 协商行为。
// Code 由业务定义，零也是合法成功码。
func SendSuccess(c fiber.Ctx, obj any, code int64) error {
	if c.Accepts(PBContentType) != "" {
		if msg, ok := obj.(proto.Message); ok {
			return SendPBSuccess(c, msg, code)
		}
	}
	return c.JSON(&AjaxResponse{Code: code, Payload: obj, Success: true})
}

// SendError 按调用方指定的 HTTP 状态发送明确可公开的消息。
func SendError(c fiber.Ctx, message string, code int64, statusCode int) error {
	return SendErrorWithRedirect(c, message, "", code, statusCode)
}

// SendErrorWithRedirect 发送公开错误及可选跳转地址，不改写调用方状态码。
func SendErrorWithRedirect(c fiber.Ctx, message, redirect string, code int64, statusCode int) error {
	if !strings.Contains(c.Get(fiber.HeaderAccept), "*/*") && c.Accepts(PBContentType) != "" {
		return SendPBError(c, message, code, statusCode)
	}
	return c.Status(statusCode).JSON(&AjaxResponse{Code: code, Message: message, Redirect: redirect})
}

// SendErrors 仅公开明确分类的业务消息；未知、系统、数据库或混合错误使用安全文案。
// 包装或组合错误中的内部错误优先，不泄露底层错误文本；HTTP 与业务状态码保持不变。
func SendErrors(c fiber.Ctx, err error, code int64, statusCode int) error {
	return SendError(c, publicErrorMessage(err), code, statusCode)
}

func publicErrorMessage(err error) string {
	const internal = "系统内部错误"
	var message string
	unsafe := false
	visited := 0
	var visit func(error, int)
	visit = func(current error, depth int) {
		if current == nil {
			return
		}
		visited++
		if depth > 64 || visited > 1024 {
			unsafe = true
			return
		}
		messageNode := false
		if classified, ok := current.(baseerrors.Error); ok {
			if baseerrors.IsSystemError(classified) || baseerrors.IsDBError(classified) {
				unsafe = true
				return
			}
			if baseerrors.IsMessageError(classified) {
				messageNode = true
				if message == "" {
					message = classified.Error()
				}
			}
		}
		switch wrapped := current.(type) {
		case interface{ Unwrap() []error }:
			children := wrapped.Unwrap()
			if len(children) == 0 && !messageNode {
				unsafe = true
			}
			for _, child := range children {
				visit(child, depth+1)
			}
		case interface{ Unwrap() error }:
			child := wrapped.Unwrap()
			if child == nil && !messageNode {
				unsafe = true
			}
			visit(child, depth+1)
		default:
			if !messageNode {
				unsafe = true
			}
		}
	}
	visit(err, 0)
	if unsafe || message == "" {
		return internal
	}
	return message
}
