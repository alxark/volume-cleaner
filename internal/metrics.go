package internal

import (
	"log/slog"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	srv *http.Server
	mux *http.ServeMux
	lis net.Listener
	log *slog.Logger
}

func NewMetrics(addr string, log *slog.Logger) (m *Metrics, err error) {
	if addr == `` {
		addr = `127.0.0.1:23456`
	}

	lis, err := net.Listen(`tcp`, addr)
	if err != nil {
		return m, err
	}

	m = &Metrics{
		lis: lis,
		mux: http.NewServeMux(),
		log: log,
		srv: &http.Server{
			Addr: addr,
		},
	}
	m.mux.Handle("/metrics", promhttp.Handler())

	return m, nil
}

func (m *Metrics) Run() (err error) {
	m.srv.Handler = m.mux

	m.log.Info("starting metrics service", "addr", m.srv.Addr)
	return m.srv.Serve(m.lis)
}
