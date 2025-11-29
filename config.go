package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	defaultPort = "9090"
)

// parseBatteries parses battery configuration from environment variables
func parseBatteries() ([]Battery, error) {
	ips := os.Getenv("SONNENBATTERIE_IPS")
	if ips == "" {
		return nil, fmt.Errorf("SONNENBATTERIE_IPS must be set")
	}

	tokens := os.Getenv("SONNENBATTERIE_TOKENS")
	if tokens == "" {
		return nil, fmt.Errorf("SONNENBATTERIE_TOKENS must be set")
	}

	ipList := strings.Split(ips, ",")
	tokenList := strings.Split(tokens, ",")
	names := strings.Split(os.Getenv("SONNENBATTERIE_NAMES"), ",")

	if len(ipList) != len(tokenList) {
		return nil, fmt.Errorf("number of IPs (%d) must match number of tokens (%d)", len(ipList), len(tokenList))
	}

	batteries := make([]Battery, 0, len(ipList))
	for i := range ipList {
		ip := strings.TrimSpace(ipList[i])
		token := strings.TrimSpace(tokenList[i])
		if ip == "" || token == "" {
			continue
		}

		name := "battery" + strconv.Itoa(i)
		if i < len(names) && strings.TrimSpace(names[i]) != "" {
			name = strings.TrimSpace(names[i])
		}

		batteries = append(batteries, Battery{
			Name:      name,
			IP:        ip,
			AuthToken: token,
		})
	}

	if len(batteries) == 0 {
		return nil, fmt.Errorf("no valid batteries configured")
	}

	return batteries, nil
}

// getPort returns the configured port or the default
func getPort() string {
	port := os.Getenv("EXPORTER_PORT")
	if port == "" {
		port = defaultPort
	}
	return port
}
