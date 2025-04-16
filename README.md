# Network Monitor

A simple Go application to monitor network traffic on a specified interface, calculate the throughput, and send notifications to Discord via webhook if a configurable threshold is exceeded. It also reports the top IP addresses contributing to the traffic.

## Prerequisites

*   **Go:** Version 1.18 or later (for building the project).
*   **libpcap development libraries:** Required for packet capturing.
    *   On Debian/Ubuntu: `sudo apt-get update && sudo apt-get install libpcap-dev`
    *   On Fedora/CentOS/RHEL: `sudo dnf install libpcap-devel`
    *   On macOS (with Homebrew): `brew install libpcap` (usually included with Xcode Command Line Tools)

## Configuration

Configuration can be provided via a YAML file, environment variables, or command-line flags. The order of precedence is: Flags > Environment Variables > Config File > Defaults.

**Configuration File:**

By default, the application looks for `config.yaml` in the current directory, `$HOME/.config/network-monitor/`, or `/etc/network-monitor/`. You can specify a different path using the `--config` flag.

Example `config.yaml`:

```yaml
# Network interface to capture packets from.
# Leave empty ("") to let the program automatically select the first non-loopback interface.
# Example: "eth0", "enp3s0"
interface: ""

# Network speed threshold in Mbps.
# If the overall network speed exceeds this value, a notification is sent.
threshold_mbps: 100.0

# Discord webhook URL to send notifications to.
# If empty, notifications are disabled.
webhook_url: "https://discord.com/api/webhooks/..."

# Time interval in seconds for checking the network speed and sending reports.
interval_seconds: 60

# Number of top IP addresses (by traffic volume) to include in the notification.
top_n: 5
```

**Environment Variables:**

Set environment variables prefixed with `NM_`.

*   `NM_INTERFACE`
*   `NM_THRESHOLD_MBPS`
*   `NM_WEBHOOK_URL`
*   `NM_INTERVAL_SECONDS`
*   `NM_TOP_N`

Example: `export NM_THRESHOLD_MBPS=150.5`

**Command-line Flags:**

*   `--config`: Path to the configuration file.
*   `--interface`: Network interface name.
*   `--threshold_mbps`: Speed threshold in Mbps.
*   `--webhook_url`: Discord webhook URL.
*   `--interval_seconds`: Monitoring interval in seconds.
*   `--top_n`: Number of top talkers to report.

Example: `./monitor --interface eth0 --threshold_mbps 200`

## Build Instructions

1.  Clone the repository (if you haven't already).
2.  Navigate to the project root directory.
3.  Build the binary:
    ```bash
    go build -o monitor ./cmd/monitor
    ```
    This will create an executable file named `monitor` in the current directory.

## Usage Instructions

Running the monitor requires privileges to capture network packets.

**Option 1: Run as root** (Simplest, but less secure)

```bash
sudo ./monitor --config /path/to/your/config.yaml
```
*(Replace `/path/to/your/config.yaml` if you are not using one of the default locations)*

**Option 2: Grant capabilities** (More secure)

Grant the necessary capabilities to the executable:

```bash
sudo setcap cap_net_raw,cap_net_admin=eip ./monitor
```

Then run the monitor as a regular user:

```bash
./monitor --config /path/to/your/config.yaml
```

The application will then start monitoring the specified network interface and send notifications according to the configuration.

## Example Discord Notification

When the traffic exceeds the configured threshold, a notification similar to this will be sent to the specified Discord webhook URL:

```
----------------------------------------
🚨 Network Threshold Exceeded! 🚨
----------------------------------------
📊 Current Speed: 125.67 Mbps (Threshold: 100.00 Mbps)
⏰ Time: 2024-07-28 15:30:00 UTC

🔝 Top 5 Talkers (IP: Bytes):
1.  192.168.1.100: 5.2 GB
2.  10.0.0.5: 2.1 GB
3.  8.8.8.8: 800.5 MB
4.  172.16.10.20: 450.1 MB
5.  1.1.1.1: 300.0 MB
----------------------------------------
```
*(Note: The exact format might vary slightly depending on implementation details.)* 