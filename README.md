# Network Monitor

A simple Go application to monitor network traffic speed on a specified interface and optionally send alerts via webhook.

## Showcase
![image](https://github.com/user-attachments/assets/b2a08258-e394-41ad-863c-c4a9a4726c34)
![image](https://github.com/user-attachments/assets/1b532925-176f-4c3c-b33e-a86c34a684c5)

## Features

*   Monitors network traffic speed (upload/download).
*   Compares current speed against a configurable threshold (in Mbps).
*   Reports monitoring results at a regular interval.
*   Identifies top N network talkers (based on bytes transferred).
*   Optional webhook integration for alerts when the threshold is exceeded.
*   Prometheus metrics endpoint for monitoring and alerting.
*   Configuration via a YAML file (`config.yaml`), environment variables, or command-line flags.

## Prerequisites

*   Go 1.24 or later installed ([Go Installation Guide](https://golang.org/doc/install)).
*   `libpcap` development libraries (needed for `gopacket`).
    *   On Debian/Ubuntu: `sudo apt-get update && sudo apt-get install libpcap-dev`
    *   On Fedora/CentOS/RHEL: `sudo dnf install libpcap-devel`
    *   On macOS (with Homebrew): `brew install libpcap`

## Installation

1.  **Clone the repository:**
    ```bash
    git clone <your-repository-url>
    cd network-monitor
    ```
2.  **Build the application:**
    ```bash
    go build -o network-monitor ./cmd/monitor
    ```
    This will create the `network-monitor` executable in the current directory.

## Configuration

The application uses [Viper](https://github.com/spf13/viper) for configuration management. Configuration can be provided through:

1.  A `config.yaml` file in the same directory as the executable (or in `$HOME/.network-monitor/`, or `/etc/network-monitor/`).
2.  Environment variables (prefixed with `NM_`, e.g., `NM_INTERFACE_NAME=eth0`).
3.  Command-line flags (though currently commented out in `main.go`, Viper might still support them depending on setup).

A `config.yaml.example` file is provided. Copy it to `config.yaml` and modify it according to your needs:

```bash
cp config.yaml.example config.yaml
```

**Key Configuration Options:**

*   `interface_name`: The network interface to monitor (e.g., `eth0`, `en0`). If empty, the application attempts to find the first non-loopback interface.
*   `threshold_mbps`: The speed threshold in Megabits per second (Mbps).
*   `interval_seconds`: The monitoring interval in seconds.
*   `webhook_url`: (Optional) The URL to send a POST request to when the threshold is exceeded.
*   `top_n`: The number of top talkers (IP addresses) to report based on traffic volume during the interval.
*   `metrics_enabled`: Whether to enable the Prometheus metrics endpoint (default: true).
*   `metrics_port`: The port on which to expose the Prometheus metrics (default: "9090").

See `internal/config/config.go` and `config.yaml.example` for all options.

## Usage

Run the compiled binary:

```bash
./network-monitor
```

The application will load the configuration and start monitoring the specified network interface. Press `Ctrl+C` to stop the monitor gracefully.

Ensure the application has the necessary permissions to capture network traffic. You might need to run it with `sudo` or set appropriate capabilities:

```bash
sudo setcap cap_net_raw,cap_net_admin=eip ./network-monitor
# Then run without sudo
./network-monitor
```

*(Adjust `setcap` command based on your specific OS and security practices)*

## Docker Support

This application can be run as a Docker container using Docker Compose.

### Prerequisites

- Docker and Docker Compose installed on your system.

### Running with Docker Compose

1. **Create a configuration file**:
   ```bash
   cp config.yaml.example config.yaml
   ```
   Edit the `config.yaml` file to adjust settings as needed.

2. **Start the container**:
   ```bash
   docker compose up -d
   ```

3. **View logs**:
   ```bash
   docker compose logs -f
   ```

4. **Stop the container**:
   ```bash
   docker compose down
   ```

### Configuration with Docker

When running in Docker, the application uses host networking (`network_mode: "host"`) to access the network interfaces directly. This means:

- The network interface specified should be the host's interface name
- Prometheus metrics will be exposed on the host's network at port 9090 (by default)

You can configure the application using either:
- The `config.yaml` file mounted into the container
- Environment variables in the `.env` file
- Environment variables passed directly to the `docker compose` command:
  ```bash
  INTERFACE_NAME=eth0 docker compose up -d
  ```

## Prometheus Integration

The application exposes metrics through a Prometheus endpoint that can be used for monitoring and alerting. By default, the metrics are exposed at `http://localhost:9090/metrics`.

### Available Metrics

* `network_speed_mbps` - Current network speed in Mbps
* `network_traffic_bytes_total` - Total network traffic in bytes
* `network_top_talkers_mbps` - Top network talkers by speed in Mbps
* `network_threshold_exceeded` - Whether the network speed threshold is exceeded (1 for yes, 0 for no)

### Prometheus Configuration

To scrape these metrics with Prometheus, add the following to your `prometheus.yml` configuration:

```yaml
scrape_configs:
  - job_name: 'network-monitor'
    static_configs:
      - targets: ['localhost:9090']
```

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

(Optional: Add your license information here, e.g., MIT License)
