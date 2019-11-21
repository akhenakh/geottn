package geottn

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	MsgReceivedCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "geottn",
			Name:      "received_msg_total",
			Help:      "The total number of received msg from TTN",
		},
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
