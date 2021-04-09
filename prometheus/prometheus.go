package prometheus

import (
	"strings"
	"time"

	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/configs"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	conf             configs.PrometheusConfig
	requestsTotal    *prometheus.CounterVec
	requestsCosttime *prometheus.SummaryVec
	float64Duration  = float64(time.Millisecond)
)

func Init(c configs.PrometheusConfig) {
	if c.IsEnable == false {
		return
	}
	conf = c
	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: c.RequestsTotal,
			Help: strings.ReplaceAll(c.RequestsTotal, "_", " "),
		},
		[]string{"app", "host", "path", "method"},
	)
	requestsCosttime = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       c.RequestsCosttime,
			Help:       strings.ReplaceAll(c.RequestsCosttime, "_", " "),
			Objectives: map[float64]float64{0.5: 0.05, 0.75: 0.05, 0.9: 0.01, 0.99: 0.001},
			// MaxAge:     time.Minute,
		},
		[]string{"app", "host", "path", "method"},
	)
}

func RequestsTotal(path, method string) {
	if conf.IsEnable == false {
		return
	}
	requestsTotal.WithLabelValues(common.GetAppName(),
		common.GetHostName(),
		path,
		method).Inc()
}

func RequestsCosttime(path, method string, duration time.Duration) {
	if conf.IsEnable == false {
		return
	}
	requestsCosttime.WithLabelValues(common.GetAppName(),
		common.GetHostName(),
		path,
		method).Observe(float64(duration) / float64Duration)
}
