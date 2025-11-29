package main

import (
	"os"
	"testing"
)

func TestParseBatteries(t *testing.T) {
	tests := []struct {
		name          string
		envIPs        string
		envTokens     string
		envNames      string
		wantCount     int
		wantFirstName string
		wantFirstIP   string
		wantErr       bool
	}{
		{
			name:          "single battery with name",
			envIPs:        "192.168.1.100",
			envTokens:     "token123",
			envNames:      "home",
			wantCount:     1,
			wantFirstName: "home",
			wantFirstIP:   "192.168.1.100",
			wantErr:       false,
		},
		{
			name:          "single battery without name",
			envIPs:        "192.168.1.100",
			envTokens:     "token123",
			envNames:      "",
			wantCount:     1,
			wantFirstName: "battery0",
			wantFirstIP:   "192.168.1.100",
			wantErr:       false,
		},
		{
			name:      "multiple batteries",
			envIPs:    "192.168.1.100,192.168.1.101",
			envTokens: "token1,token2",
			envNames:  "house,garage",
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "multiple batteries with spaces",
			envIPs:    " 192.168.1.100 , 192.168.1.101 ",
			envTokens: " token1 , token2 ",
			envNames:  " house , garage ",
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "mismatched IP and token count",
			envIPs:    "192.168.1.100,192.168.1.101",
			envTokens: "token1",
			envNames:  "",
			wantErr:   true,
		},
		{
			name:      "missing IPs",
			envIPs:    "",
			envTokens: "token1",
			envNames:  "",
			wantErr:   true,
		},
		{
			name:      "missing tokens",
			envIPs:    "192.168.1.100",
			envTokens: "",
			envNames:  "",
			wantErr:   true,
		},
		{
			name:      "empty values skipped",
			envIPs:    "192.168.1.100,,192.168.1.101",
			envTokens: "token1,,token2",
			envNames:  "house,,garage",
			wantCount: 2,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			_ = os.Setenv("SONNENBATTERIE_IPS", tt.envIPs)
			_ = os.Setenv("SONNENBATTERIE_TOKENS", tt.envTokens)
			_ = os.Setenv("SONNENBATTERIE_NAMES", tt.envNames)
			defer func() {
				_ = os.Unsetenv("SONNENBATTERIE_IPS")
				_ = os.Unsetenv("SONNENBATTERIE_TOKENS")
				_ = os.Unsetenv("SONNENBATTERIE_NAMES")
			}()

			batteries, err := parseBatteries()

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseBatteries() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("parseBatteries() unexpected error: %v", err)
				return
			}

			if len(batteries) != tt.wantCount {
				t.Errorf("parseBatteries() got %d batteries, want %d", len(batteries), tt.wantCount)
				return
			}

			if tt.wantCount > 0 {
				if tt.wantFirstName != "" && batteries[0].Name != tt.wantFirstName {
					t.Errorf("first battery name = %s, want %s", batteries[0].Name, tt.wantFirstName)
				}
				if tt.wantFirstIP != "" && batteries[0].IP != tt.wantFirstIP {
					t.Errorf("first battery IP = %s, want %s", batteries[0].IP, tt.wantFirstIP)
				}
			}
		})
	}
}

func TestGetPort(t *testing.T) {
	tests := []struct {
		name    string
		envPort string
		want    string
	}{
		{
			name:    "default port",
			envPort: "",
			want:    "9090",
		},
		{
			name:    "custom port",
			envPort: "8080",
			want:    "8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envPort != "" {
				_ = os.Setenv("EXPORTER_PORT", tt.envPort)
				defer func() { _ = os.Unsetenv("EXPORTER_PORT") }()
			}

			got := getPort()
			if got != tt.want {
				t.Errorf("getPort() = %s, want %s", got, tt.want)
			}
		})
	}
}
