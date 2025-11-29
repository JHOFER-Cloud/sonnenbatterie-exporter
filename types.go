package main

// Battery represents a single SonnenBatterie instance
type Battery struct {
	Name      string
	IP        string
	AuthToken string
}

// ICStatus contains internal component status information
type ICStatus struct {
	StateBMS               string `json:"statebms"`
	StateCoreControlModule string `json:"statecorecontrolmodule"`
	StateInverter          string `json:"stateinverter"`
	NrBatteryModules       int    `json:"nrbatterymodules"`
}

// LatestData represents the response from /api/v2/latestdata
// This endpoint combines status and system information
type LatestData struct {
	ConsumptionW       float64  `json:"Consumption_W"`
	FullChargeCapacity int      `json:"FullChargeCapacity"`
	GridFeedInW        float64  `json:"GridFeedIn_W"`
	PacTotalW          float64  `json:"Pac_total_W"`
	ProductionW        float64  `json:"Production_W"`
	RSOC               int      `json:"RSOC"` // Relative State of Charge
	USOC               int      `json:"USOC"` // User State of Charge
	Timestamp          string   `json:"Timestamp"`
	ICStatus           ICStatus `json:"ic_status"`
}

// Status represents the response from /api/v2/status
// This endpoint provides additional real-time status information
type Status struct {
	BatteryCharging    bool    `json:"BatteryCharging"`
	BatteryDischarging bool    `json:"BatteryDischarging"`
	SystemStatus       string  `json:"SystemStatus"`
	Uac                float64 `json:"Uac"`  // AC Voltage
	Ubat               float64 `json:"Ubat"` // Battery Voltage
	Fac                float64 `json:"Fac"`  // AC Frequency
}
