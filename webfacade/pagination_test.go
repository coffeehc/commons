package webfacade

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestStrictPaginationCountOptInAndOpaqueCursor(t *testing.T) {
	for _, test := range []struct {
		query      string
		mode       PaginationMode
		wantCount  bool
		wantCursor string
	}{
		{"page_index=0&page_size=20", OffsetPagination, false, ""},
		{"page_index=0&page_size=20&return_total=false", OffsetPagination, false, ""},
		{"page_index=0&page_size=20&return_total=true", OffsetPagination, true, ""},
		{"page_size=20", CursorPagination, false, ""},
		{"page_size=20&return_total=true&cursor=" + url.QueryEscape(" opaque:AA+/= "), CursorPagination, true, " opaque:AA+/= "},
	} {
		t.Run(test.query[:min(len(test.query), 100)], func(t *testing.T) {
			app := fiber.New(fiber.Config{ReadBufferSize: 16384})
			var parsed PaginationQuery
			var parseErr error
			app.Get("/", func(c fiber.Ctx) error {
				parsed, parseErr = ParsePaginationQuery(c, test.mode, 200)
				return c.SendStatus(204)
			})
			testQueryRequest(t, app, test.query)
			if parseErr != nil || parsed.ReturnTotal != test.wantCount || parsed.Cursor != test.wantCursor || parsed.PageSize != 20 {
				t.Fatalf("%+v %v", parsed, parseErr)
			}
		})
	}
}

func TestStrictPaginationRejectsMalformedAndMixedModes(t *testing.T) {
	for _, test := range []struct {
		query string
		mode  PaginationMode
	}{
		{"page_index=0", OffsetPagination}, {"page_size=20", OffsetPagination},
		{"page_index=-1&page_size=20", OffsetPagination}, {"page_index=01&page_size=20", OffsetPagination},
		{"page_index=0&page_size=0", OffsetPagination}, {"page_index=0&page_size=201", OffsetPagination},
		{"page_index=0&page_size=20&return_total=1", OffsetPagination}, {"page_index=0&page_size=20&return_total=", OffsetPagination},
		{"page_index=0&page_size=20&return_total=true&return_total=false", OffsetPagination},
		{"page_index=9007199254740991&page_size=20", OffsetPagination},
		{"page_index=0&page_size=20&cursor=opaque", OffsetPagination},
		{"page_size=20&page_index=0", CursorPagination}, {"page_size=20&cursor=", CursorPagination},
		{"page_size=20&cursor=" + strings.Repeat("x", MaxCursorBytes+1), CursorPagination},
		{"page_size=20&cursor=%ZZ", CursorPagination}, {"page_size=20", PaginationMode("unknown")},
	} {
		t.Run(test.query[:min(len(test.query), 100)], func(t *testing.T) {
			app := fiber.New(fiber.Config{ReadBufferSize: 16384})
			var parseErr error
			app.Get("/", func(c fiber.Ctx) error {
				_, parseErr = ParsePaginationQuery(c, test.mode, 200)
				return c.SendStatus(204)
			})
			testQueryRequest(t, app, test.query)
			if parseErr == nil {
				t.Fatal("invalid pagination accepted")
			}
		})
	}
}

func testQueryRequest(t *testing.T, app *fiber.App, query string) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/?"+query, http.NoBody)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
}
