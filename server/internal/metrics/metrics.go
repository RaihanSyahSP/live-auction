package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry          *prometheus.Registry
	bidsTotal         *prometheus.CounterVec
	bidDuration       prometheus.Histogram
	broadcastDuration prometheus.Histogram
	connectedClients  *prometheus.GaugeVec
	liveAuctions      prometheus.Gauge
	loginTotal        *prometheus.CounterVec
	refreshTotal      *prometheus.CounterVec
}

func New() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		bidsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auction_bids_total",
				Help: "Total bid requests by result.",
			},
			[]string{"result"},
		),
		bidDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "auction_bid_duration_seconds",
				Help:    "Bid service duration.",
				Buckets: prometheus.DefBuckets,
			},
		),
		broadcastDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "auction_broadcast_duration_seconds",
				Help:    "WebSocket broadcast duration.",
				Buckets: prometheus.DefBuckets,
			},
		),
		connectedClients: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "auction_connected_clients",
				Help: "Connected WebSocket clients per auction.",
			},
			[]string{"auction_id"},
		),
		liveAuctions: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "auction_live_auctions",
				Help: "Number of live auctions.",
			},
		),
		loginTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auth_login_total",
				Help: "Login attempts by result.",
			},
			[]string{"result"},
		),
		refreshTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auth_refresh_total",
				Help: "Refresh attempts by result.",
			},
			[]string{"result"},
		),
	}

	m.registry.MustRegister(
		m.bidsTotal,
		m.bidDuration,
		m.broadcastDuration,
		m.connectedClients,
		m.liveAuctions,
		m.loginTotal,
		m.refreshTotal,
	)

	return m
}

func (m *Metrics) Handler() http.Handler {
	if m == nil {
		return http.NotFoundHandler()
	}
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Metrics) ObserveBid(result string, duration time.Duration) {
	if m == nil {
		return
	}
	m.bidsTotal.WithLabelValues(result).Inc()
	m.bidDuration.Observe(duration.Seconds())
}

func (m *Metrics) ObserveBroadcast(duration time.Duration) {
	if m == nil {
		return
	}
	m.broadcastDuration.Observe(duration.Seconds())
}

func (m *Metrics) SetConnectedClients(auctionID string, count int) {
	if m == nil {
		return
	}
	m.connectedClients.WithLabelValues(auctionID).Set(float64(count))
}

func (m *Metrics) SetLiveAuctions(count int) {
	if m == nil {
		return
	}
	m.liveAuctions.Set(float64(count))
}

func (m *Metrics) ObserveLogin(result string) {
	if m == nil {
		return
	}
	m.loginTotal.WithLabelValues(result).Inc()
}

func (m *Metrics) ObserveRefresh(result string) {
	if m == nil {
		return
	}
	m.refreshTotal.WithLabelValues(result).Inc()
}
