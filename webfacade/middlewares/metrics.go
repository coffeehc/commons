package middlewares

import (
	"github.com/coffeehc/boot/plugin/manage/metrics"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"time"
)

// var reqCnt = &Metric{
//   ID:          "reqCnt",
//   Name:        "requests_total",
//   Description: "How many HTTP requests processed, partitioned by status code and HTTP method.",
//   Type:        "counter_vec",
//   Args:        []string{"code", "method", "handler", "host", "url"}}
//
// var reqDur = &Metric{
//   ID:          "reqDur",
//   Name:        "request_duration_seconds",
//   Description: "The HTTP request latencies in seconds.",
//   Type:        "histogram_vec",
//   Args:        []string{"code", "method", "url"},
// }
//
// var resSz = &Metric{
//   ID:          "resSz",
//   Name:        "response_size_bytes",
//   Description: "The HTTP response sizes in bytes.",
//   Type:        "summary"}
//
// var reqSz = &Metric{
//   ID:          "reqSz",
//   Name:        "request_size_bytes",
//   Description: "The HTTP request sizes in bytes.",
//   Type:        "summary"}
//

func MetricsMiddleware() gin.HandlerFunc {
	reqCount := metrics.NewCounterVec(metrics.CollectorOpt{
		Namespace: "http",
		Subsystem: "web",
		Name:      "requests_total",
	}, []string{"url", "status_code", "method"})
	reqDur := metrics.NewHistogramVec(metrics.CollectorOpt{
		Namespace: "http",
		Subsystem: "web",
		Name:      "request_dur",
	}, []float64{
		float64(10),
		float64(50),
		float64(100),
		float64(500),
		float64(1000),
		float64(5000),
		float64(10000),
		float64(30000),
	}, []string{"url", "status_code", "method"})
	// reqDur1 := metrics.NewSummaryVec(metrics.CollectorOpt{
	//   Namespace: "http",
	//   Subsystem: "web",
	//   Name:      "request_dur",
	// }, time.Minute*5, 10, 100, []string{"url","status_code","method"})
	metrics.RegisterMetrics(reqCount)
	metrics.RegisterMetrics(reqDur)
	// metrics.RegisterMetrics(reqDur1)
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		status := strconv.Itoa(c.Writer.Status())
		elapsed := float64(time.Since(start)) / float64(time.Millisecond)
		url := c.Request.URL.Path
		labels := prometheus.Labels{
			"url":         url,
			"status_code": status,
			"method":      c.Request.Method,
		}
		reqCount.With(labels).Inc()
		reqDur.With(labels).Observe(elapsed)
		// reqDur1.With(labels).Observe(elapsed)
	}
}
