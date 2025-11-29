package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// Mock API server that simulates sonnenBatterie-api responses
func newMockAPIServer() *httptest.Server {
	mux := http.NewServeMux()

	// Mock /api/system endpoint
	mux.HandleFunc("/api/system", func(w http.ResponseWriter, r *http.Request) {
		system := System{
			IP:              "10.0.0.100",
			WanIP:           "31.31.31.31",
			Model:           "eco 8.0",
			MAC:             "AA:BB:CC:DD:EE:FF",
			SoftwareVersion: "1.8.3",
			HardwareVersion: "1.0",
			LED:             "green",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(system)
	})

	// Mock /api/consumption endpoint
	mux.HandleFunc("/api/consumption", func(w http.ResponseWriter, r *http.Request) {
		consumption := Consumption{
			CurrentMW: 1500000, // 1.5 kW
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(consumption)
	})

	// Mock /api/status endpoint
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		status := Status{
			GridFeedInMW: 500000,  // 0.5 kW feed-in
			ProductionMW: 3000000, // 3 kW production
			ChargeLevel:  85,
			ChargeMode:   "charging",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	})

	return httptest.NewServer(mux)
}

func TestParseBatteries_SingleURL(t *testing.T) {
	os.Setenv("SONNENBATTERIE_API_URL", "http://test:8080")
	defer os.Unsetenv("SONNENBATTERIE_API_URL")

	batteries, err := parseBatteries()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(batteries) != 1 {
		t.Fatalf("Expected 1 battery, got %d", len(batteries))
	}

	if batteries[0].APIURL != "http://test:8080" {
		t.Errorf("Expected URL http://test:8080, got %s", batteries[0].APIURL)
	}

	if batteries[0].Name != "battery0" {
		t.Errorf("Expected name battery0, got %s", batteries[0].Name)
	}
}

func TestParseBatteries_MultipleURLs(t *testing.T) {
	os.Setenv("SONNENBATTERIE_API_URLS", "http://test1:8080,http://test2:8080")
	os.Setenv("SONNENBATTERIE_NAMES", "house,garage")
	defer os.Unsetenv("SONNENBATTERIE_API_URLS")
	defer os.Unsetenv("SONNENBATTERIE_NAMES")

	batteries, err := parseBatteries()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(batteries) != 2 {
		t.Fatalf("Expected 2 batteries, got %d", len(batteries))
	}

	if batteries[0].Name != "house" {
		t.Errorf("Expected name house, got %s", batteries[0].Name)
	}

	if batteries[1].Name != "garage" {
		t.Errorf("Expected name garage, got %s", batteries[1].Name)
	}
}

func TestParseBatteries_NoURLs(t *testing.T) {
	os.Unsetenv("SONNENBATTERIE_API_URL")
	os.Unsetenv("SONNENBATTERIE_API_URLS")

	_, err := parseBatteries()
	if err == nil {
		t.Fatal("Expected error when no URLs configured, got nil")
	}
}

func TestParseBatteries_WithSpaces(t *testing.T) {
	os.Setenv("SONNENBATTERIE_API_URLS", " http://test1:8080 , http://test2:8080 ")
	defer os.Unsetenv("SONNENBATTERIE_API_URLS")

	batteries, err := parseBatteries()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(batteries) != 2 {
		t.Fatalf("Expected 2 batteries, got %d", len(batteries))
	}

	// Check that spaces were trimmed
	if batteries[0].APIURL != "http://test1:8080" {
		t.Errorf("Expected URL without spaces, got %s", batteries[0].APIURL)
	}
}

func TestCollector_Integration(t *testing.T) {
	// Start mock API server
	server := newMockAPIServer()
	defer server.Close()

	// Create collector with mock server
	batteries := []Battery{
		{
			Name:   "test",
			APIURL: server.URL,
		},
	}
	collector := NewCollector(batteries)

	// Register collector
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	// Collect metrics
	metricCount := testutil.CollectAndCount(collector)
	if metricCount == 0 {
		t.Fatal("Expected metrics to be collected, got 0")
	}

	// Verify specific metrics exist
	expectedMetrics := []string{
		"sonnenbatterie_charge_level_percent",
		"sonnenbatterie_consumption_mw",
		"sonnenbatterie_production_mw",
		"sonnenbatterie_grid_feed_in_mw",
		"sonnenbatterie_charging",
		"sonnenbatterie_discharging",
		"sonnenbatterie_info",
		"sonnenbatterie_scrape_success",
	}

	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	foundMetrics := make(map[string]bool)
	for _, m := range metrics {
		foundMetrics[m.GetName()] = true
	}

	for _, expected := range expectedMetrics {
		if !foundMetrics[expected] {
			t.Errorf("Expected metric %s not found", expected)
		}
	}

	// Verify metric values by checking the gathered metrics
	var chargeLevelFound bool
	for _, m := range metrics {
		if m.GetName() == "sonnenbatterie_charge_level_percent" {
			chargeLevelFound = true
			if len(m.GetMetric()) > 0 {
				value := m.GetMetric()[0].GetGauge().GetValue()
				if value != 85 { // Expected value from mock
					t.Errorf("Expected charge level to be 85, got %f", value)
				}
			}
		}
	}
	if !chargeLevelFound {
		t.Error("Expected charge level metric to be present")
	}
}

func TestCollector_FailedScrape(t *testing.T) {
	// Create collector with invalid URL
	batteries := []Battery{
		{
			Name:   "test",
			APIURL: "http://invalid-host-that-does-not-exist:8080",
		},
	}
	collector := NewCollector(batteries)

	// Register collector
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	// Collect metrics - should not panic
	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Find scrape_success metric
	var scrapeSuccessFound bool
	for _, m := range metrics {
		if m.GetName() == "sonnenbatterie_scrape_success" {
			scrapeSuccessFound = true
			// Should be 0 (failed)
			if len(m.GetMetric()) > 0 {
				value := m.GetMetric()[0].GetGauge().GetValue()
				if value != 0 {
					t.Errorf("Expected scrape_success to be 0 (failed), got %f", value)
				}
			}
		}
	}

	if !scrapeSuccessFound {
		t.Error("Expected scrape_success metric to be present")
	}
}

func TestCollector_MultipleBatteries(t *testing.T) {
	// Start two mock API servers
	server1 := newMockAPIServer()
	defer server1.Close()

	server2 := newMockAPIServer()
	defer server2.Close()

	batteries := []Battery{
		{Name: "battery1", APIURL: server1.URL},
		{Name: "battery2", APIURL: server2.URL},
	}

	collector := NewCollector(batteries)
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	// Collect metrics
	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Verify we have metrics for both batteries
	batteryNames := make(map[string]bool)
	for _, m := range metrics {
		for _, metric := range m.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "battery_name" {
					batteryNames[label.GetValue()] = true
				}
			}
		}
	}

	if !batteryNames["battery1"] {
		t.Error("Expected metrics for battery1")
	}
	if !batteryNames["battery2"] {
		t.Error("Expected metrics for battery2")
	}
}

func TestFetchJSON_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"test": "value"})
	}))
	defer server.Close()

	var result map[string]string
	err := fetchJSON(server.URL, &result)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result["test"] != "value" {
		t.Errorf("Expected value 'value', got %s", result["test"])
	}
}

func TestFetchJSON_InvalidURL(t *testing.T) {
	var result map[string]string
	err := fetchJSON("http://invalid-host:8080/test", &result)
	if err == nil {
		t.Fatal("Expected error for invalid URL, got nil")
	}
}

func TestFetchJSON_Non200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	var result map[string]string
	err := fetchJSON(server.URL, &result)
	if err == nil {
		t.Fatal("Expected error for 404 status, got nil")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Errorf("Expected error to mention status code 404, got: %v", err)
	}
}

func TestFetchJSON_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	var result map[string]string
	err := fetchJSON(server.URL, &result)
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
}

func TestChargeMode_Charging(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/api/status") {
			status := Status{
				GridFeedInMW: 0,
				ProductionMW: 0,
				ChargeLevel:  50,
				ChargeMode:   "charging",
			}
			json.NewEncoder(w).Encode(status)
		} else if strings.HasSuffix(r.URL.Path, "/api/system") {
			system := System{Model: "test", MAC: "00:00:00:00:00:00"}
			json.NewEncoder(w).Encode(system)
		} else if strings.HasSuffix(r.URL.Path, "/api/consumption") {
			consumption := Consumption{CurrentMW: 0}
			json.NewEncoder(w).Encode(consumption)
		}
	}))
	defer server.Close()

	collector := NewCollector([]Battery{{Name: "test", APIURL: server.URL}})
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	// Gather and check charging metric
	metrics, _ := registry.Gather()
	var chargingValue float64 = -1
	var dischargingValue float64 = -1

	for _, m := range metrics {
		if m.GetName() == "sonnenbatterie_charging" && len(m.GetMetric()) > 0 {
			chargingValue = m.GetMetric()[0].GetGauge().GetValue()
		}
		if m.GetName() == "sonnenbatterie_discharging" && len(m.GetMetric()) > 0 {
			dischargingValue = m.GetMetric()[0].GetGauge().GetValue()
		}
	}

	if chargingValue != 1.0 {
		t.Errorf("Expected charging to be 1.0, got %f", chargingValue)
	}
	if dischargingValue != 0.0 {
		t.Errorf("Expected discharging to be 0.0, got %f", dischargingValue)
	}
}

func TestChargeMode_Discharging(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/api/status") {
			status := Status{
				GridFeedInMW: 0,
				ProductionMW: 0,
				ChargeLevel:  50,
				ChargeMode:   "discharging",
			}
			json.NewEncoder(w).Encode(status)
		} else if strings.HasSuffix(r.URL.Path, "/api/system") {
			system := System{Model: "test", MAC: "00:00:00:00:00:00"}
			json.NewEncoder(w).Encode(system)
		} else if strings.HasSuffix(r.URL.Path, "/api/consumption") {
			consumption := Consumption{CurrentMW: 0}
			json.NewEncoder(w).Encode(consumption)
		}
	}))
	defer server.Close()

	collector := NewCollector([]Battery{{Name: "test", APIURL: server.URL}})
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	metrics, _ := registry.Gather()
	var chargingValue float64 = -1
	var dischargingValue float64 = -1

	for _, m := range metrics {
		if m.GetName() == "sonnenbatterie_charging" && len(m.GetMetric()) > 0 {
			chargingValue = m.GetMetric()[0].GetGauge().GetValue()
		}
		if m.GetName() == "sonnenbatterie_discharging" && len(m.GetMetric()) > 0 {
			dischargingValue = m.GetMetric()[0].GetGauge().GetValue()
		}
	}

	if chargingValue != 0.0 {
		t.Errorf("Expected charging to be 0.0, got %f", chargingValue)
	}
	if dischargingValue != 1.0 {
		t.Errorf("Expected discharging to be 1.0, got %f", dischargingValue)
	}
}
