package redisstream

import "github.com/prometheus/client_golang/prometheus"

var (
	redisStreamOpsBuckets = prometheus.LinearBuckets(0.001, 0.005, 20)

	sendMessageLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "RedisStream__Send__Message__Latency",
		Help:    "Latency of sending a message in second",
		Buckets: redisStreamOpsBuckets,
	})
	readMessageLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "RedisStream__Read__Message__Latency",
		Help:    "Latency of a blocking group read in second",
		Buckets: redisStreamOpsBuckets,
	})

	sendMessageTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "RedisStream__Send__Message__Total",
		Help: "Number of messages produced",
	})
	readMessageTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "RedisStream__Read__Message__Total",
		Help: "Number of messages delivered to the consumer channel",
	})
	ackMessageTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "RedisStream__Ack__Message__Total",
		Help: "Number of messages acknowledged",
	})
	claimMessageTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "RedisStream__Claim__Message__Total",
		Help: "Number of messages reclaimed after exceeding max processing time",
	})

	RedisStreamMetricsCollectors = []prometheus.Collector{
		sendMessageLatency,
		sendMessageTotal,

		readMessageLatency,
		readMessageTotal,
		ackMessageTotal,
		claimMessageTotal,
	}
)
