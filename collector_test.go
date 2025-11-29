package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewCollector(t *testing.T) {
	batteries := []Battery{
		{Name: "test1", IP: "192.168.1.100", AuthToken: "token1"},
		{Name: "test2", IP: "192.168.1.101", AuthToken: "token2"},
	}

	collector := NewCollector(batteries)

	if len(collector.batteries) != 2 {
		t.Errorf("NewCollector() batteries count = %d, want 2", len(collector.batteries))
	}

	if collector.chargeLevel == nil {
		t.Error("NewCollector() chargeLevel metric is nil")
	}

	if collector.userChargeLevel == nil {
		t.Error("NewCollector() userChargeLevel metric is nil")
	}

	if collector.batteryPower == nil {
		t.Error("NewCollector() batteryPower metric is nil")
	}

	if collector.fullChargeCapacity == nil {
		t.Error("NewCollector() fullChargeCapacity metric is nil")
	}
}

func TestCollector_Describe(t *testing.T) {
	batteries := []Battery{
		{Name: "test", IP: "192.168.1.100", AuthToken: "token"},
	}

	collector := NewCollector(batteries)
	descCh := make(chan *prometheus.Desc, 20)

	go func() {
		collector.Describe(descCh)
		close(descCh)
	}()

	count := 0
	for range descCh {
		count++
	}

	// We have 14 metrics: chargeLevel, userChargeLevel, consumption, production, gridFeedIn,
	// batteryPower, fullChargeCapacity, charging, discharging, acVoltage, batteryVoltage,
	// acFrequency, info, scrapeSuccess
	expectedCount := 14
	if count != expectedCount {
		t.Errorf("Describe() sent %d descriptors, want %d", count, expectedCount)
	}
}

func TestCollector_Collect_EmptyBatteries(t *testing.T) {
	collector := NewCollector([]Battery{})
	metricCh := make(chan prometheus.Metric, 100)

	go func() {
		collector.Collect(metricCh)
		close(metricCh)
	}()

	count := 0
	for range metricCh {
		count++
	}

	if count != 0 {
		t.Errorf("Collect() with no batteries sent %d metrics, want 0", count)
	}
}

func TestCollector_Collect_Success(t *testing.T) {
	// Create mock servers for latestdata and status endpoints
	mockLatestData := LatestData{
		ConsumptionW:       750.5,
		FullChargeCapacity: 5000,
		GridFeedInW:        -250.0,
		PacTotalW:          100.0,
		ProductionW:        500.0,
		RSOC:               85,
		USOC:               83,
		Timestamp:          "2025-11-29 21:00:00",
		ICStatus: ICStatus{
			StateBMS:               "ready",
			StateCoreControlModule: "ongrid",
			StateInverter:          "running",
			NrBatteryModules:       2,
		},
	}

	mockStatus := Status{
		BatteryCharging:    true,
		BatteryDischarging: false,
		ConsumptionW:       750.5,
		GridFeedInW:        -250.0,
		PacTotalW:          100.0,
		ProductionW:        500.0,
		SystemStatus:       "OnGrid",
		Uac:                230.0,
		Ubat:               50.0,
		Fac:                50.0,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Auth-Token") != "test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v2/latestdata":
			_ = json.NewEncoder(w).Encode(mockLatestData)
		case "/api/v2/status":
			_ = json.NewEncoder(w).Encode(mockStatus)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	battery := Battery{
		Name:      "test-battery",
		IP:        server.URL[7:], // Remove "http://" prefix
		AuthToken: "test-token",
	}

	collector := NewCollector([]Battery{battery})
	metricCh := make(chan prometheus.Metric, 100)

	go func() {
		collector.Collect(metricCh)
		close(metricCh)
	}()

	// Count metrics and verify we got all of them
	count := 0
	for range metricCh {
		count++
	}

	// We expect: scrapeSuccess + chargeLevel + userChargeLevel + consumption + production +
	// gridFeedIn + batteryPower + fullChargeCapacity + charging + discharging + acVoltage +
	// batteryVoltage + acFrequency + info = 14 metrics
	expectedCount := 14
	if count != expectedCount {
		t.Errorf("Collect() sent %d metrics, want %d", count, expectedCount)
	}
}

func TestCollector_Collect_LatestDataError(t *testing.T) {
	// Create a server that returns error for latestdata
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/latestdata" {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	battery := Battery{
		Name:      "test-battery",
		IP:        server.URL[7:],
		AuthToken: "test-token",
	}

	collector := NewCollector([]Battery{battery})
	metricCh := make(chan prometheus.Metric, 100)

	go func() {
		collector.Collect(metricCh)
		close(metricCh)
	}()

	// Should only get scrapeSuccess metric with value 0
	count := 0
	for range metricCh {
		count++
	}

	if count != 1 {
		t.Errorf("Collect() with latestdata error sent %d metrics, want 1 (scrapeSuccess)", count)
	}
}

func TestCollector_Collect_StatusError(t *testing.T) {
	// Create a server that returns success for latestdata but error for status
	mockLatestData := LatestData{
		ConsumptionW:       750.5,
		FullChargeCapacity: 5000,
		GridFeedInW:        -250.0,
		PacTotalW:          100.0,
		ProductionW:        500.0,
		RSOC:               85,
		USOC:               83,
		Timestamp:          "2025-11-29 21:00:00",
		ICStatus: ICStatus{
			StateBMS:               "ready",
			StateCoreControlModule: "ongrid",
			StateInverter:          "running",
			NrBatteryModules:       2,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v2/latestdata":
			_ = json.NewEncoder(w).Encode(mockLatestData)
		case "/api/v2/status":
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	battery := Battery{
		Name:      "test-battery",
		IP:        server.URL[7:],
		AuthToken: "test-token",
	}

	collector := NewCollector([]Battery{battery})
	metricCh := make(chan prometheus.Metric, 100)

	go func() {
		collector.Collect(metricCh)
		close(metricCh)
	}()

	// Should only get scrapeSuccess metric with value 0
	count := 0
	for range metricCh {
		count++
	}

	if count != 1 {
		t.Errorf("Collect() with status error sent %d metrics, want 1 (scrapeSuccess)", count)
	}
}

func TestCollector_Collect_MultipleBatteries(t *testing.T) {
	// Create mock data
	mockLatestData := LatestData{
		ConsumptionW:       750.5,
		FullChargeCapacity: 5000,
		GridFeedInW:        -250.0,
		PacTotalW:          100.0,
		ProductionW:        500.0,
		RSOC:               85,
		USOC:               83,
		Timestamp:          "2025-11-29 21:00:00",
		ICStatus: ICStatus{
			StateBMS:               "ready",
			StateCoreControlModule: "ongrid",
			StateInverter:          "running",
			NrBatteryModules:       2,
		},
	}

	mockStatus := Status{
		BatteryCharging:    false,
		BatteryDischarging: true,
		ConsumptionW:       750.5,
		GridFeedInW:        -250.0,
		PacTotalW:          100.0,
		ProductionW:        500.0,
		SystemStatus:       "OnGrid",
		Uac:                230.0,
		Ubat:               50.0,
		Fac:                50.0,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v2/latestdata":
			_ = json.NewEncoder(w).Encode(mockLatestData)
		case "/api/v2/status":
			_ = json.NewEncoder(w).Encode(mockStatus)
		}
	}))
	defer server.Close()

	batteries := []Battery{
		{Name: "battery1", IP: server.URL[7:], AuthToken: "token1"},
		{Name: "battery2", IP: server.URL[7:], AuthToken: "token2"},
	}

	collector := NewCollector(batteries)
	metricCh := make(chan prometheus.Metric, 100)

	go func() {
		collector.Collect(metricCh)
		close(metricCh)
	}()

	count := 0
	for range metricCh {
		count++
	}

	// 14 metrics per battery * 2 batteries = 28 metrics
	expectedCount := 28
	if count != expectedCount {
		t.Errorf("Collect() with 2 batteries sent %d metrics, want %d", count, expectedCount)
	}
}
