package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultPort = "9090"
)

// API response structures from sonnenBatterie-api
type System struct {
	IP              string `json:"ip"`
	WanIP           string `json:"wanIp"`
	Model           string `json:"model"`
	MAC             string `json:"mac"`
	SoftwareVersion string `json:"softwareVersion"`
	HardwareVersion string `json:"hardwareVersion"`
	LED             string `json:"led"`
}

type Consumption struct {
	CurrentMW int `json:"current_mw"`
}

type Status struct {
	GridFeedInMW int    `json:"grid_feed_in_mw"`
	ProductionMW int    `json:"production_mw"`
	ChargeLevel  int    `json:"charge_level"`
	ChargeMode   string `json:"charge_mode"`
}

// Battery represents a single SonnenBatterie instance
type Battery struct {
	Name   string
	APIURL string
}

// Collector implements prometheus.Collector
type Collector struct {
	batteries []Battery

	// Metrics
	chargeLevel   *prometheus.Desc
	consumption   *prometheus.Desc
	production    *prometheus.Desc
	gridFeedIn    *prometheus.Desc
	charging      *prometheus.Desc
	discharging   *prometheus.Desc
	info          *prometheus.Desc
	scrapeSuccess *prometheus.Desc
}

// NewCollector creates a new SonnenBatterie collector
func NewCollector(batteries []Battery) *Collector {
	return &Collector{
		batteries: batteries,
		chargeLevel: prometheus.NewDesc(
			"sonnenbatterie_charge_level_percent",
			"Battery charge level in percent",
			[]string{"battery_name", "model", "mac"},
			nil,
		),
		consumption: prometheus.NewDesc(
			"sonnenbatterie_consumption_mw",
			"Current house consumption in milliwatts",
			[]string{"battery_name", "model", "mac"},
			nil,
		),
		production: prometheus.NewDesc(
			"sonnenbatterie_production_mw",
			"Current solar production in milliwatts",
			[]string{"battery_name", "model", "mac"},
			nil,
		),
		gridFeedIn: prometheus.NewDesc(
			"sonnenbatterie_grid_feed_in_mw",
			"Current grid feed-in in milliwatts",
			[]string{"battery_name", "model", "mac"},
			nil,
		),
		charging: prometheus.NewDesc(
			"sonnenbatterie_charging",
			"Battery is currently charging (1=yes, 0=no)",
			[]string{"battery_name", "model", "mac"},
			nil,
		),
		discharging: prometheus.NewDesc(
			"sonnenbatterie_discharging",
			"Battery is currently discharging (1=yes, 0=no)",
			[]string{"battery_name", "model", "mac"},
			nil,
		),
		info: prometheus.NewDesc(
			"sonnenbatterie_info",
			"SonnenBatterie system information",
			[]string{"battery_name", "model", "mac", "sw_version", "hw_version", "led", "ip", "wan_ip"},
			nil,
		),
		scrapeSuccess: prometheus.NewDesc(
			"sonnenbatterie_scrape_success",
			"Whether scraping the battery API was successful",
			[]string{"battery_name"},
			nil,
		),
	}
}

// Describe implements prometheus.Collector
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.chargeLevel
	ch <- c.consumption
	ch <- c.production
	ch <- c.gridFeedIn
	ch <- c.charging
	ch <- c.discharging
	ch <- c.info
	ch <- c.scrapeSuccess
}

// Collect implements prometheus.Collector
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	var wg sync.WaitGroup

	for _, battery := range c.batteries {
		wg.Add(1)
		go func(b Battery) {
			defer wg.Done()
			c.collectBattery(b, ch)
		}(battery)
	}

	wg.Wait()
}

func (c *Collector) collectBattery(battery Battery, ch chan<- prometheus.Metric) {
	// Fetch all data from the battery
	system, err := fetchSystem(battery.APIURL)
	if err != nil {
		log.Printf("Error fetching system info for %s: %v", battery.Name, err)
		ch <- prometheus.MustNewConstMetric(c.scrapeSuccess, prometheus.GaugeValue, 0, battery.Name)
		return
	}

	consumption, err := fetchConsumption(battery.APIURL)
	if err != nil {
		log.Printf("Error fetching consumption for %s: %v", battery.Name, err)
		ch <- prometheus.MustNewConstMetric(c.scrapeSuccess, prometheus.GaugeValue, 0, battery.Name)
		return
	}

	status, err := fetchStatus(battery.APIURL)
	if err != nil {
		log.Printf("Warning: Could not fetch status for %s (will use partial metrics): %v", battery.Name, err)
		status = nil // Mark as unavailable
	}

	// Mark as successful if we got system and consumption data
	ch <- prometheus.MustNewConstMetric(c.scrapeSuccess, prometheus.GaugeValue, 1, battery.Name)

	// Common labels
	labels := []string{battery.Name, system.Model, system.MAC}

	// Always emit consumption metric (from consumption endpoint)
	ch <- prometheus.MustNewConstMetric(c.consumption, prometheus.GaugeValue, float64(consumption.CurrentMW), labels...)

	// Emit status-dependent metrics only if available
	if status != nil {
		ch <- prometheus.MustNewConstMetric(c.chargeLevel, prometheus.GaugeValue, float64(status.ChargeLevel), labels...)
		ch <- prometheus.MustNewConstMetric(c.production, prometheus.GaugeValue, float64(status.ProductionMW), labels...)
		ch <- prometheus.MustNewConstMetric(c.gridFeedIn, prometheus.GaugeValue, float64(status.GridFeedInMW), labels...)

		// Charge mode as binary metrics
		charging := 0.0
		if status.ChargeMode == "charging" {
			charging = 1.0
		}
		discharging := 0.0
		if status.ChargeMode == "discharging" {
			discharging = 1.0
		}
		ch <- prometheus.MustNewConstMetric(c.charging, prometheus.GaugeValue, charging, labels...)
		ch <- prometheus.MustNewConstMetric(c.discharging, prometheus.GaugeValue, discharging, labels...)
	} else {
		log.Printf("Status endpoint unavailable for %s, some metrics will be missing", battery.Name)
	}

	// System info
	infoLabels := []string{
		battery.Name,
		system.Model,
		system.MAC,
		system.SoftwareVersion,
		system.HardwareVersion,
		system.LED,
		system.IP,
		system.WanIP,
	}
	ch <- prometheus.MustNewConstMetric(c.info, prometheus.GaugeValue, 1, infoLabels...)
}

func fetchSystem(apiURL string) (*System, error) {
	var system System
	if err := fetchJSON(apiURL+"/api/system", &system); err != nil {
		return nil, err
	}
	return &system, nil
}

func fetchConsumption(apiURL string) (*Consumption, error) {
	var consumption Consumption
	if err := fetchJSON(apiURL+"/api/consumption", &consumption); err != nil {
		return nil, err
	}
	return &consumption, nil
}

func fetchStatus(apiURL string) (*Status, error) {
	var status Status
	if err := fetchJSON(apiURL+"/api/status", &status); err != nil {
		return nil, err
	}
	return &status, nil
}

func fetchJSON(url string, target interface{}) error {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d from %s", resp.StatusCode, url)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode JSON from %s: %w", url, err)
	}

	return nil
}

func parseBatteries() ([]Battery, error) {
	apiURLs := os.Getenv("SONNENBATTERIE_API_URLS")
	if apiURLs == "" {
		apiURLs = os.Getenv("SONNENBATTERIE_API_URL") // Fallback to single URL
		if apiURLs == "" {
			return nil, fmt.Errorf("SONNENBATTERIE_API_URLS or SONNENBATTERIE_API_URL must be set")
		}
	}

	urls := strings.Split(apiURLs, ",")
	names := strings.Split(os.Getenv("SONNENBATTERIE_NAMES"), ",")

	batteries := make([]Battery, 0, len(urls))
	for i, url := range urls {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}

		name := "battery" + strconv.Itoa(i)
		if i < len(names) && strings.TrimSpace(names[i]) != "" {
			name = strings.TrimSpace(names[i])
		}

		batteries = append(batteries, Battery{
			Name:   name,
			APIURL: url,
		})
	}

	if len(batteries) == 0 {
		return nil, fmt.Errorf("no valid battery URLs configured")
	}

	return batteries, nil
}

func main() {
	port := os.Getenv("EXPORTER_PORT")
	if port == "" {
		port = defaultPort
	}

	// Parse battery configurations
	batteries, err := parseBatteries()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	log.Printf("Starting SonnenBatterie Prometheus Exporter on port %s", port)
	log.Printf("Monitoring %d battery/batteries:", len(batteries))
	for _, b := range batteries {
		log.Printf("  - %s: %s", b.Name, b.APIURL)
	}

	// Create and register collector
	collector := NewCollector(batteries)
	prometheus.MustRegister(collector)

	// Expose metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
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
			batteriesList.WriteString(fmt.Sprintf("<li>%s: %s</li>\n", b.Name, b.APIURL))
		}
		fmt.Fprintf(w, html, len(batteries), batteriesList.String())
	})

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
