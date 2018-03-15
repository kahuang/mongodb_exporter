package collector

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rwynn/gtm"
	"gopkg.in/mgo.v2"
)

var (
	oplogEntryCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "oplogtail",
		Name:      "entry_count",
		Help:      "The total number of entries observed in the oplog by ns/op",
	}, []string{"ns", "op"})
	oplogTailError = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "oplogtail",
		Name:      "tail_error",
		Help:      "The total number of errors while tailing the oplog",
	})
)

var tailer *OplogTailStats

type OplogTailStats struct{}

func (o *OplogTailStats) Start(session *mgo.Session) {
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)

	ctx := gtm.Start(session, nil)

	// ctx.OpC is a channel to read ops from
	// ctx.ErrC is a channel to read errors from
	// ctx.Stop() stops all go routines started by gtm.Start
	for {
		// loop forever receiving events
		select {
		case err := <-ctx.ErrC:
			// handle errors
			fmt.Println(err)
		case op := <-ctx.OpC:
			oplogEntryCount.WithLabelValues(op.Namespace, op.Operation).Add(1)
		}
	}
}

// Export exports metrics to Prometheus
func (status *OplogTailStats) Export(ch chan<- prometheus.Metric) {
	oplogEntryCount.Collect(ch)
	oplogTailError.Collect(ch)
}

// Describe describes metrics collected
func (status *OplogTailStats) Describe(ch chan<- *prometheus.Desc) {
	oplogEntryCount.Describe(ch)
	oplogTailError.Describe(ch)
}

func GetOplogTailStats(session *mgo.Session) *OplogTailStats {
	if tailer == nil {
		tailer = &OplogTailStats{}
		// Start a tailer with a copy of the session (to avoid messing with the other metrics in the session)
		tailer.Start(session.Copy())
	}

	return tailer
}