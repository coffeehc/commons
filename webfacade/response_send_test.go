package webfacade

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	baseerrors "github.com/coffeehc/base/errors"
	"github.com/gofiber/fiber/v3"
	"google.golang.org/protobuf/proto"
)

func testResponse(t *testing.T, accept string, handler fiber.Handler) (int, []byte) {
	t.Helper()
	app := fiber.New()
	app.Get("/", handler)
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Accept", accept)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, body
}

func TestSendSuccessCodeZeroAndJSONBytes(t *testing.T) {
	status, body := testResponse(t, "application/json", func(c fiber.Ctx) error { return SendSuccess(c, []string{}, 0) })
	if status != 200 || string(body) != `{"success":true,"message":"","payload":[],"request_id":"","code":0}` {
		t.Fatalf("%d %s", status, body)
	}
}

func TestErrorMessagesNeverExposeInternalCauses(t *testing.T) {
	const secret = "password=DO-NOT-EXPOSE"
	public := baseerrors.MessageError("invalid request")
	for name, err := range map[string]error{
		"plain": errors.New(secret), "system": baseerrors.SystemError(secret),
		"database":                baseerrors.BuildError(baseerrors.ErrorSystemDB, secret),
		"wrapped-system":          fmt.Errorf("context: %w", baseerrors.SystemError(secret)),
		"message-then-system":     errors.Join(public, baseerrors.SystemError(secret)),
		"system-then-message":     errors.Join(baseerrors.SystemError(secret), public),
		"message-wrapping-system": wrappedPublicError{public, baseerrors.SystemError(secret)},
		"message-then-plain":      errors.Join(public, errors.New(secret)),
		"nil":                     nil,
	} {
		t.Run(name, func(t *testing.T) {
			for _, accept := range []string{"application/json", "application/x-protobuf", "*/*"} {
				status, body := testResponse(t, accept, func(c fiber.Ctx) error { return SendErrors(c, err, 50301, 503) })
				if status != 503 || strings.Contains(string(body), secret) || !strings.Contains(string(body), "系统内部错误") {
					t.Fatalf("%d %q", status, body)
				}
				if accept == "application/x-protobuf" {
					var response PBResponse
					if e := proto.Unmarshal(body, &response); e != nil || response.Success || response.Code != 50301 {
						t.Fatalf("%+v %v", &response, e)
					}
				}
			}
		})
	}
	for _, err := range []error{public, fmt.Errorf("private wrapper: %w", public)} {
		_, body := testResponse(t, "application/json", func(c fiber.Ctx) error { return SendErrors(c, err, 40001, 400) })
		if !strings.Contains(string(body), "invalid request") || strings.Contains(string(body), "private wrapper") {
			t.Fatalf("%s", body)
		}
	}
}

func TestUnifiedPBErrorsPreserveHTTPStatus(t *testing.T) {
	for _, handler := range []fiber.Handler{
		func(c fiber.Ctx) error { return SendPBError(c, "denied", 40301, 403) },
		func(c fiber.Ctx) error { return SendPBErrors(c, baseerrors.MessageError("denied"), 40301, 403) },
		func(c fiber.Ctx) error { return SendError(c, "denied", 40301, 403) },
	} {
		status, body := testResponse(t, "application/x-protobuf", handler)
		var response PBResponse
		err := proto.Unmarshal(body, &response)
		if err != nil || status != 403 || response.Success || response.Code != 40301 || response.Message != "denied" {
			t.Fatalf("%d %+v %v", status, &response, err)
		}
	}
}

func TestRedirectAndRequestIDEnvelope(t *testing.T) {
	status, body := testResponse(t, "application/json", func(c fiber.Ctx) error { return SendErrorWithRedirect(c, "sign in", "/login", 40101, 401) })
	if status != 401 || string(body) != `{"success":false,"message":"sign in","payload":null,"request_id":"","code":40101,"redirect":"/login"}` {
		t.Fatalf("%d %s", status, body)
	}
}

// A custom classified error may also have a cause; system causes still win.
type classifiedPublicError interface{ baseerrors.Error }

type wrappedPublicError struct {
	classifiedPublicError
	cause error
}

func (e wrappedPublicError) Unwrap() error { return e.cause }
