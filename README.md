# SonnenBatterie Prometheus Exporter

Prometheus exporter for SonnenBatterie that talks directly to the battery's native API.

[![CI/CD](https://github.com/JHOFER-Cloud/sonnenbatterie-exporter/actions/workflows/ci-cd.yml/badge.svg)](https://github.com/JHOFER-Cloud/sonnenbatterie-exporter/actions/workflows/ci-cd.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

## Features

- **Direct API Integration**: Talks directly to SonnenBatterie's native `/api/v2/latestdata` and `/api/v2/status` endpoints
- **Multi-Battery Support**: Monitor multiple batteries from a single exporter instance
- **Rich Metrics**: Exports 11 comprehensive metrics including charge levels, power flow, and system status
- **Health Labels**: Includes BMS state and inverter state labels for enhanced monitoring
- **Production Ready**: Comprehensive test coverage, linting, and CI/CD pipeline

## Quick Start

### Kubernetes (Recommended)

**Single Battery:**

1. Create a secret for your battery credentials:

   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: sonnenbatterie-credentials
     namespace: monitoring
   type: Opaque
   stringData:
     auth-token: "your-auth-token-here"
   ```

2. Deploy the exporter:
   ```bash
   kubectl apply -f k8s/deployment.yaml
   ```

**Multiple Batteries:**

1. Edit `k8s/deployment-multi-battery.yaml` to set each battery's IP and auth token

2. Deploy:
   ```bash
   kubectl apply -f k8s/deployment-multi-battery.yaml
   ```

### Docker

**Single Battery:**

```bash
docker run -d -p 9090:9090 \
  -e SONNENBATTERIE_IPS="192.168.1.100" \
  -e SONNENBATTERIE_TOKENS="your-auth-token" \
  -e SONNENBATTERIE_NAMES="home" \
  ghcr.io/jhofer-cloud/sonnenbatterie-exporter:latest
```

**Multiple Batteries:**

```bash
docker run -d -p 9090:9090 \
  -e SONNENBATTERIE_IPS="192.168.1.100,192.168.1.101" \
  -e SONNENBATTERIE_TOKENS="token1,token2" \
  -e SONNENBATTERIE_NAMES="house,garage" \
  ghcr.io/jhofer-cloud/sonnenbatterie-exporter:latest
```

## Configuration

| Variable                | Description                                   | Required | Default |
| ----------------------- | --------------------------------------------- | -------- | ------- |
| `SONNENBATTERIE_IPS`    | Comma-separated battery IP addresses          | Yes      | -       |
| `SONNENBATTERIE_TOKENS` | Comma-separated Auth-Token values             | Yes      | -       |
| `SONNENBATTERIE_NAMES`  | Comma-separated battery names (optional)      | No       | battery0, battery1, ... |
| `EXPORTER_PORT`         | Metrics port                                  | No       | 9090    |

**Notes:**
- The number of IPs and tokens must match
- Names are optional - if not provided, batteries will be named `battery0`, `battery1`, etc.
- Empty values in comma-separated lists are skipped (e.g., `"ip1,,ip3"` is valid)

## Authentication

The exporter uses the SonnenBatterie's Auth-Token for authentication. To get your token:

1. Log in to your SonnenBatterie web interface
2. Navigate to the API settings or security section
3. Generate or copy your Auth-Token
4. Use this token in the `SONNENBATTERIE_TOKENS` environment variable

The exporter sends the token via the `Auth-Token` HTTP header when making requests to the battery.

## Metrics

All metrics include these labels:
- `battery_name` - Name of the battery (from `SONNENBATTERIE_NAMES` or auto-generated)
- `bms_state` - Battery Management System state (e.g., "ready")
- `inverter_state` - Inverter state (e.g., "running")

### Gauge Metrics

- `sonnenbatterie_charge_level_percent` - Battery charge level (RSOC) (0-100%)
- `sonnenbatterie_user_charge_level_percent` - User-visible charge level (USOC) (0-100%)
- `sonnenbatterie_battery_power_mw` - Battery power (negative = charging, positive = discharging) (milliwatts)
- `sonnenbatterie_full_charge_capacity_wh` - Full charge capacity (watt-hours)
- `sonnenbatterie_consumption_mw` - House consumption (milliwatts)
- `sonnenbatterie_production_mw` - Solar production (milliwatts)
- `sonnenbatterie_grid_feed_in_mw` - Grid feed-in/consumption (milliwatts, negative = consuming from grid)
- `sonnenbatterie_ac_voltage` - AC voltage (volts)
- `sonnenbatterie_battery_voltage` - Battery voltage (volts)
- `sonnenbatterie_ac_frequency` - AC frequency (hertz)

### Info Metrics

- `sonnenbatterie_system_info` - System information with labels:
  - `battery_name` - Battery name
  - `system_status` - System status (e.g., "OnGrid")
  - `bms_state` - BMS state
  - `core_control_module_state` - Core control module state
  - `inverter_state` - Inverter state
  - `charging` - Whether battery is charging (true/false)
  - `discharging` - Whether battery is discharging (true/false)
  - `battery_modules` - Number of battery modules

## API Endpoints

The exporter uses these SonnenBatterie API endpoints:

- `/api/v2/latestdata` - Latest battery data (charge, power, production, consumption, etc.)
- `/api/v2/status` - Current status (charging state, voltages, frequency)

## Development

```bash
# Run tests
just test

# Run linter
just lint

# Build
just build

# Run locally (requires battery on network)
export SONNENBATTERIE_IPS="192.168.1.100"
export SONNENBATTERIE_TOKENS="your-token"
just run
```

### Code Structure

- `main.go` - Entry point and HTTP server setup
- `types.go` - Data structures for battery API responses
- `client.go` - HTTP client for battery API
- `config.go` - Environment variable parsing
- `collector.go` - Prometheus metrics collector
- `*_test.go` - Comprehensive test suite

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.
