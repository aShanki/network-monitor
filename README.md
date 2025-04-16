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

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

(Optional: Add your license information here, e.g., MIT License)
