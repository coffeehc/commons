package webfacade

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestParseOrderConditionsAcceptsOnlyValidDirections(t *testing.T) {
	app := fiber.New()
	var names []string
	var orders []string
	app.Get("/", func(c fiber.Ctx) error {
		conditions := ParseOrderConditions(c, map[string]FieldDefined{
			"name":    {RealFieldName: "display_name"},
			"created": {RealFieldName: "created_at"},
			"invalid": {RealFieldName: "invalid_field"},
		})
		for _, condition := range conditions {
			names = append(names, condition.Name)
			orders = append(orders, condition.Order)
		}
		return c.SendStatus(fiber.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet,
		"/?sort=name,created,invalid,missing&sort_name=asc&sort_created=desc&sort_invalid=sideways",
		http.NoBody)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if len(names) != 2 || names[0] != "display_name" || names[1] != "created_at" {
		t.Fatalf("unexpected order fields: %v", names)
	}
	if orders[0] != "asc" || orders[1] != "desc" {
		t.Fatalf("unexpected order directions: %v", orders)
	}
}
