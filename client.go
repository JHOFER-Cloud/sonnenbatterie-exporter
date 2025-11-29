package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// fetchLatestData retrieves the latest data from a SonnenBatterie
func fetchLatestData(battery Battery) (*LatestData, error) {
	var data LatestData
	url := fmt.Sprintf("http://%s/api/v2/latestdata", battery.IP)
	if err := fetchJSON(url, battery.AuthToken, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// fetchStatus retrieves the current status from a SonnenBatterie
func fetchStatus(battery Battery) (*Status, error) {
	var status Status
	url := fmt.Sprintf("http://%s/api/v2/status", battery.IP)
	if err := fetchJSON(url, battery.AuthToken, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// fetchJSON performs an HTTP GET request with authentication and decodes the JSON response
func fetchJSON(url string, token string, target interface{}) error {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for %s: %w", url, err)
	}
	req.Header.Set("Auth-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d from %s", resp.StatusCode, url)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode JSON from %s: %w", url, err)
	}

	return nil
}
