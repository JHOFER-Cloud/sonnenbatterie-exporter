package main

import (
	"log"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Collector implements prometheus.Collector for SonnenBatterie metrics
type Collector struct {
	batteries []Battery

	// Metrics
	chargeLevel        *prometheus.Desc
	userChargeLevel    *prometheus.Desc
	consumption        *prometheus.Desc
	production         *prometheus.Desc
	gridFeedIn         *prometheus.Desc
	batteryPower       *prometheus.Desc
	charging           *prometheus.Desc
	discharging        *prometheus.Desc
	fullChargeCapacity *prometheus.Desc
	info               *prometheus.Desc
	scrapeSuccess      *prometheus.Desc
}

// NewCollector creates a new SonnenBatterie collector
func NewCollector(batteries []Battery) *Collector {
	return &Collector{
		batteries: batteries,
		chargeLevel: prometheus.NewDesc(
			"sonnenbatterie_charge_level_percent",
			"Battery relative state of charge (RSOC) in percent",
			[]string{"battery_name", "bms_state", "inverter_state"},
			nil,
		),
		userChargeLevel: prometheus.NewDesc(
			"sonnenbatterie_user_charge_level_percent",
			"Battery user state of charge (USOC) in percent",
			[]string{"battery_name", "bms_state", "inverter_state"},
			nil,
		),
		consumption: prometheus.NewDesc(
			"sonnenbatterie_consumption_mw",
			"Current house consumption in milliwatts",
			[]string{"battery_name", "bms_state", "inverter_state"},
			nil,
		),
		production: prometheus.NewDesc(
			"sonnenbatterie_production_mw",
			"Current solar production in milliwatts",
			[]string{"battery_name", "bms_state", "inverter_state"},
			nil,
		),
		gridFeedIn: prometheus.NewDesc(
			"sonnenbatterie_grid_feed_in_mw",
			"Current grid feed-in in milliwatts (negative=consuming)",
			[]string{"battery_name", "bms_state", "inverter_state"},
			nil,
		),
		batteryPower: prometheus.NewDesc(
			"sonnenbatterie_battery_power_mw",
			"Current battery power in milliwatts (positive=charging, negative=discharging)",
			[]string{"battery_name", "bms_state", "inverter_state"},
			nil,
		),
		charging: prometheus.NewDesc(
			"sonnenbatterie_charging",
			"Battery is currently charging (1=yes, 0=no)",
			[]string{"battery_name", "bms_state", "inverter_state"},
			nil,
		),
		discharging: prometheus.NewDesc(
			"sonnenbatterie_discharging",
			"Battery is currently discharging (1=yes, 0=no)",
			[]string{"battery_name", "bms_state", "inverter_state"},
			nil,
		),
		fullChargeCapacity: prometheus.NewDesc(
			"sonnenbatterie_full_charge_capacity_wh",
			"Battery full charge capacity in watt-hours",
			[]string{"battery_name", "bms_state", "inverter_state"},
			nil,
		),
		info: prometheus.NewDesc(
			"sonnenbatterie_info",
			"SonnenBatterie system information",
			[]string{"battery_name", "bms_state", "core_control_state", "inverter_state", "battery_modules", "ip"},
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
	ch <- c.userChargeLevel
	ch <- c.consumption
	ch <- c.production
	ch <- c.gridFeedIn
	ch <- c.batteryPower
	ch <- c.charging
	ch <- c.discharging
	ch <- c.fullChargeCapacity
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
	// Fetch latest data from the battery (combines status + system info)
	latestData, err := fetchLatestData(battery)
	if err != nil {
		log.Printf("Error fetching latest data for %s: %v", battery.Name, err)
		ch <- prometheus.MustNewConstMetric(c.scrapeSuccess, prometheus.GaugeValue, 0, battery.Name)
		return
	}

	// Fetch additional status info (for charging/discharging booleans)
	status, err := fetchStatus(battery)
	if err != nil {
		log.Printf("Error fetching status for %s: %v", battery.Name, err)
		ch <- prometheus.MustNewConstMetric(c.scrapeSuccess, prometheus.GaugeValue, 0, battery.Name)
		return
	}

	// Mark as successful
	ch <- prometheus.MustNewConstMetric(c.scrapeSuccess, prometheus.GaugeValue, 1, battery.Name)

	// Common labels with state information
	labels := []string{battery.Name, latestData.ICStatus.StateBMS, latestData.ICStatus.StateInverter}

	// Emit metrics from latestdata endpoint (all in watts, convert to milliwatts)
	ch <- prometheus.MustNewConstMetric(c.chargeLevel, prometheus.GaugeValue, float64(latestData.RSOC), labels...)
	ch <- prometheus.MustNewConstMetric(c.userChargeLevel, prometheus.GaugeValue, float64(latestData.USOC), labels...)
	ch <- prometheus.MustNewConstMetric(c.consumption, prometheus.GaugeValue, latestData.ConsumptionW*1000, labels...)
	ch <- prometheus.MustNewConstMetric(c.production, prometheus.GaugeValue, latestData.ProductionW*1000, labels...)
	ch <- prometheus.MustNewConstMetric(c.gridFeedIn, prometheus.GaugeValue, latestData.GridFeedInW*1000, labels...)
	ch <- prometheus.MustNewConstMetric(c.batteryPower, prometheus.GaugeValue, latestData.PacTotalW*1000, labels...)
	ch <- prometheus.MustNewConstMetric(c.fullChargeCapacity, prometheus.GaugeValue, float64(latestData.FullChargeCapacity), labels...)

	// Charge mode as binary metrics from status endpoint
	charging := 0.0
	if status.BatteryCharging {
		charging = 1.0
	}
	discharging := 0.0
	if status.BatteryDischarging {
		discharging = 1.0
	}
	ch <- prometheus.MustNewConstMetric(c.charging, prometheus.GaugeValue, charging, labels...)
	ch <- prometheus.MustNewConstMetric(c.discharging, prometheus.GaugeValue, discharging, labels...)

	// System info
	infoLabels := []string{
		battery.Name,
		latestData.ICStatus.StateBMS,
		latestData.ICStatus.StateCoreControlModule,
		latestData.ICStatus.StateInverter,
		strconv.Itoa(latestData.ICStatus.NrBatteryModules),
		battery.IP,
	}
	ch <- prometheus.MustNewConstMetric(c.info, prometheus.GaugeValue, 1, infoLabels...)
}
