package dbmonitor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coffeehc/commons/dbsource"
	"github.com/coffeehc/commons/webfacade"
	"github.com/gofiber/fiber/v3"
)

func TestCountMonitorUsesNativeResponseEnvelope(t *testing.T) {
	monitor := NewSqlCountMonitor()
	monitor.AddRecord("SELECT 1", time.Millisecond, dbsource.HandleType(0))
	app := fiber.New()
	monitor.RegisterWebEndpoint(app)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/db/monitor/count", http.NoBody)
	request.Header.Set("Accept", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var envelope webfacade.AjaxResponse
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 || !envelope.Success || envelope.Code != 0 {
		t.Fatalf("%d %+v", response.StatusCode, envelope)
	}
	payload, ok := envelope.Payload.(map[string]any)
	if !ok || payload["SELECT 1"] != float64(1) {
		t.Fatalf("%+v", envelope.Payload)
	}
}
