package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	ViaLabel       = "via"
	ReceivedViaGW  = "GW"
	ReceivedViaTTN = "TTN"
)

var (
	MsgReceivedCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "geottn",
			Name:      "received_msg_total",
			Help:      "The total number of received msg",
		},
		[]string{ViaLabel},
	)

	ErrorCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "geottn",
			Name:      "error_total",
			Help:      "The total number of errors occurring",
		},
	)

	InsertCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "geottn",
			Name:      "insert_total",
			Help:      "The total number of inserts in db",
		},
	)
)
