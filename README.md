# SonnenBatterie Prometheus Exporter

Prometheus exporter for SonnenBatterie that works with the [sonnenBatterie-api](https://github.com/larmic-iot/sonnenBatterie-api).

[![CI/CD](https://github.com/JHOFER-Cloud/sonnenbatterie-exporter/actions/workflows/ci-cd.yml/badge.svg)](https://github.com/JHOFER-Cloud/sonnenbatterie-exporter/actions/workflows/ci-cd.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

## Architecture

This exporter requires the [sonnenBatterie-api](https://github.com/larmic-iot/sonnenBatterie-api) service to connect to your battery hardware.

```
┌─────────────────┐      ┌──────────────────────┐      ┌─────────────────┐
│ SonnenBatterie  │─────▶│ sonnenBatterie-api   │─────▶│ This Exporter   │
│ (Hardware)      │      │ (Separate Pod)       │      │ (Prometheus)    │
│ 192.168.1.100   │      │ Port 8080            │      │ Port 9090       │
└─────────────────┘      └──────────────────────┘      └─────────────────┘
                                                                │
                                                                ▼
                                                        ┌─────────────────┐
                                                        │   Prometheus    │
                                                        │   Server        │
                                                        └─────────────────┘
```

**Why two services?**

- The sonnenBatterie-api handles authentication with the hardware
- This exporter focuses on Prometheus metrics exposition
- Allows monitoring multiple batteries via multiple API instances

## Quick Start

### Kubernetes (Recommended)

**Single Battery:**

1. Edit `k8s/deployment.yaml`:

   ```yaml
   # Update your battery IP and password
   - name: SONNENBATTERIE_IP
     value: "192.168.1.100"
   - name: SONNENBATTERIE_USER_PASSWORD
     value: "your-password"
   ```

2. Deploy:
   ```bash
   kubectl apply -f k8s/deployment.yaml
   ```

**Multiple Batteries:**

1. Edit `k8s/deployment-multi-battery.yaml` to set each battery's IP and password

2. Deploy:
   ```bash
   kubectl apply -f k8s/deployment-multi-battery.yaml
   ```

This creates 2 API pods (one per battery) + 1 exporter pod monitoring both.

### Docker

**Single Battery:**

```bash
# Start the API
docker run -d -p 8080:8080 \
  -e SONNENBATTERIE_IP="192.168.1.100" \
  -e SONNENBATTERIE_USER_NAME="User" \
  -e SONNENBATTERIE_USER_PASSWORD="your-password" \
  --name sonnen-api \
  larmic/sonnen-batterie-api:latest

# Start the exporter
docker run -d -p 9090:9090 \
  -e SONNENBATTERIE_API_URL="http://sonnen-api:8080" \
  --link sonnen-api \
  ghcr.io/jhofer-cloud/sonnenbatterie-exporter:latest
```

**Multiple Batteries:**

```bash
# Start API for each battery
docker run -d -p 8081:8080 -e SONNENBATTERIE_IP="192.168.1.100" ... --name api1 ...
docker run -d -p 8082:8080 -e SONNENBATTERIE_IP="192.168.1.101" ... --name api2 ...

# Start exporter pointing to both
docker run -d -p 9090:9090 \
  -e SONNENBATTERIE_API_URLS="http://api1:8080,http://api2:8080" \
  -e SONNENBATTERIE_NAMES="house,garage" \
  --link api1 --link api2 \
  ghcr.io/jhofer-cloud/sonnenbatterie-exporter:latest
```

## Configuration

| Variable                  | Description                                       | Required |
| ------------------------- | ------------------------------------------------- | -------- |
| `SONNENBATTERIE_API_URL`  | Single API URL                                    | Yes\*    |
| `SONNENBATTERIE_API_URLS` | Comma-separated API URLs (for multiple batteries) | Yes\*    |
| `SONNENBATTERIE_NAMES`    | Comma-separated battery names                     | No       |
| `EXPORTER_PORT`           | Metrics port (default: 9090)                      | No       |

\* Use either `SONNENBATTERIE_API_URL` or `SONNENBATTERIE_API_URLS`

## Metrics

All metrics include labels: `battery_name`, `model`, `mac`

- `sonnenbatterie_charge_level_percent` - Battery charge (0-100%)
- `sonnenbatterie_consumption_mw` - House consumption (milliwatts)
- `sonnenbatterie_production_mw` - Solar production (milliwatts)
- `sonnenbatterie_grid_feed_in_mw` - Grid feed-in/consumption (milliwatts)
- `sonnenbatterie_charging` - Battery charging (0 or 1)
- `sonnenbatterie_discharging` - Battery discharging (0 or 1)
- `sonnenbatterie_info` - System information
- `sonnenbatterie_scrape_success` - Scrape health (0 or 1)

## Development

```bash
# Run tests
just test

# Build
just build

# Run locally (requires API running)
just run
```

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.
