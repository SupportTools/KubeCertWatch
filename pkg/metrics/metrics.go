package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// LastCheckTime tracks when each check was last run
	LastCheckTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "last_check_time",
		Help: "The last time the check was run",
	}, []string{"check_name"})

	// ErrorCounter tracks the number of errors encountered
	ErrorCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "certificate_check_errors_total",
		Help: "Total number of errors encountered during certificate checks",
	}, []string{"check_type", "error_type"})

	// CertificateExpiryDays tracks the days until certificate expiration
	CertificateExpiryDays = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "certificate_expiry_days",
		Help: "Days until certificate expiration",
	}, []string{"namespace", "secret_name"})
)

func init() {
	// Register all metrics
	prometheus.MustRegister(LastCheckTime)
	prometheus.MustRegister(ErrorCounter)
	prometheus.MustRegister(CertificateExpiryDays)
}
