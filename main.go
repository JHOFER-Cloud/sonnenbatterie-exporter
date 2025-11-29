package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	port := getPort()

	// Parse battery configurations
	batteries, err := parseBatteries()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	log.Printf("Starting SonnenBatterie Prometheus Exporter on port %s", port)
	log.Printf("Monitoring %d battery/batteries:", len(batteries))
	for _, b := range batteries {
		log.Printf("  - %s: %s", b.Name, b.IP)
	}

	// Create and register collector
	collector := NewCollector(batteries)
	prometheus.MustRegister(collector)

	// Expose metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Root endpoint with info
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		html := `<!DOCTYPE html>
<html>
<head><title>SonnenBatterie Exporter</title></head>
<body>
<h1>SonnenBatterie Prometheus Exporter</h1>
<p>Monitoring %d battery/batteries</p>
<ul>
%s
</ul>
<p><a href="/metrics">Metrics</a></p>
</body>
</html>`
		var batteriesList strings.Builder
		for _, b := range batteries {
			batteriesList.WriteString(fmt.Sprintf("<li>%s: %s</li>\n", b.Name, b.IP))
		}
		_, _ = fmt.Fprintf(w, html, len(batteries), batteriesList.String())
	})

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
