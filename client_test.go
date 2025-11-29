package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchLatestData(t *testing.T) {
	// Create mock response
	mockData := LatestData{
		ConsumptionW:       748,
		FullChargeCapacity: 4882,
		GridFeedInW:        -749.0,
		PacTotalW:          -1,
		ProductionW:        0,
		RSOC:               8,
		USOC:               0,
		Timestamp:          "2025-11-29 21:10:50",
		ICStatus: ICStatus{
			StateBMS:               "ready",
			StateCoreControlModule: "ongrid",
			StateInverter:          "running",
			NrBatteryModules:       1,
		},
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check auth token header
		if r.Header.Get("Auth-Token") != "test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Check endpoint
		if r.URL.Path != "/api/v2/latestdata" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockData)
	}))
	defer server.Close()

	battery := Battery{
		Name:      "test",
		IP:        server.URL[7:], // Remove "http://" prefix
		AuthToken: "test-token",
	}

	data, err := fetchLatestData(battery)
	if err != nil {
		t.Fatalf("fetchLatestData() error = %v", err)
	}

	if data.RSOC != mockData.RSOC {
		t.Errorf("RSOC = %d, want %d", data.RSOC, mockData.RSOC)
	}

	if data.ICStatus.StateBMS != mockData.ICStatus.StateBMS {
		t.Errorf("StateBMS = %s, want %s", data.ICStatus.StateBMS, mockData.ICStatus.StateBMS)
	}
}

func TestFetchStatus(t *testing.T) {
	mockStatus := Status{
		BatteryCharging:    false,
		BatteryDischarging: false,
		ConsumptionW:       750.0,
		GridFeedInW:        -250.0,
		PacTotalW:          100.0,
		ProductionW:        500.0,
		SystemStatus:       "OnGrid",
		Uac:                229.0,
		Ubat:               99.0,
		Fac:                50.016,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Auth-Token") != "test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.URL.Path != "/api/v2/status" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockStatus)
	}))
	defer server.Close()

	battery := Battery{
		Name:      "test",
		IP:        server.URL[7:],
		AuthToken: "test-token",
	}

	status, err := fetchStatus(battery)
	if err != nil {
		t.Fatalf("fetchStatus() error = %v", err)
	}

	if status.SystemStatus != mockStatus.SystemStatus {
		t.Errorf("SystemStatus = %s, want %s", status.SystemStatus, mockStatus.SystemStatus)
	}

	if status.Uac != mockStatus.Uac {
		t.Errorf("Uac = %f, want %f", status.Uac, mockStatus.Uac)
	}
}

func TestFetchJSON_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	battery := Battery{
		Name:      "test",
		IP:        server.URL[7:],
		AuthToken: "wrong-token",
	}

	_, err := fetchLatestData(battery)
	if err == nil {
		t.Error("fetchLatestData() expected error for unauthorized request")
	}
}

func TestFetchJSON_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	battery := Battery{
		Name:      "test",
		IP:        server.URL[7:],
		AuthToken: "test-token",
	}

	_, err := fetchLatestData(battery)
	if err == nil {
		t.Error("fetchLatestData() expected error for invalid JSON")
	}
}
