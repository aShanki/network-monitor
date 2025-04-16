# Network Monitor Project Plan

## 1. Project Goal

Develop a standalone network monitoring tool in Go that captures network traffic, identifies when overall speed exceeds a configurable threshold, and sends details about the top IP addresses contributing to the traffic to a Discord webhook. The application should be distributable as a single binary with minimal external dependencies.

## 2. High-Level Plan

1.  **Language & Environment Setup:** Initialize a Go project.
2.  **Packet Capture:** Implement packet capture using `gopacket`.
3.  **Traffic Analysis:** Calculate speeds and track data per source IP.
4.  **Threshold Monitoring:** Check speed against the configurable threshold.
5.  **Top Talkers Identification:** Identify top IPs when the threshold is exceeded.
6.  **Discord Integration:** Send results to a Discord webhook.
7.  **Configuration:** Allow configuration via file or flags.
8.  **Standalone Build:** Create a statically linked binary.
9.  **Error Handling & Logging:** Implement robust error/log handling.
10. **Documentation:** Write `README.md`.

## 3. Detailed Task Breakdown

### Task 3.1: Setup Go Project
*   Initialize Go module (`go mod init network-monitor`).
*   Create basic project structure (e.g., `cmd/monitor/main.go`, `internal/capture`, `internal/discord`, `internal/config`).

### Task 3.2: Implement Configuration Loading
*   Define configuration struct (Interface name, Threshold (Mbps), Webhook URL, Interval (seconds), Top N IPs).
*   Choose configuration method (e.g., Viper library for file/env/flags).
*   Load configuration at startup.

### Task 3.3: Implement Packet Capture
*   Add `gopacket` and `gopacket/pcap` dependencies.
*   Find available network interfaces or use the configured one.
*   Open the selected interface in promiscuous mode.
*   Set a BPF filter if needed (e.g., filter only IP packets).
*   Start a goroutine to read packets using `gopacket.PacketSource`.
*   Handle potential errors (permissions, interface not found).

### Task 3.4: Implement Traffic Aggregation & Speed Calculation
*   Create data structures to store packet counts and sizes per source IP within a time interval.
*   Use a ticker (e.g., `time.Ticker`) based on the configured interval.
*   In the packet reading goroutine, extract source IP and packet size (IP layer length).
*   Aggregate size per source IP for the current interval.
*   On each tick:
    *   Calculate total bytes transferred in the last interval.
    *   Calculate overall speed (e.g., `(totalBytes * 8) / intervalSeconds / 1_000_000` for Mbps).
    *   Store the per-IP data for this interval.
    *   Reset aggregators for the next interval.

### Task 3.5: Implement Threshold Logic & Top Talkers Identification
*   In the ticker goroutine, after calculating the speed:
    *   Compare the overall speed against the configured threshold.
    *   If threshold exceeded:
        *   Sort the stored per-IP data by bytes transferred (descending).
        *   Take the top N IPs.
        *   Calculate the speed for each top IP (similar to overall speed calculation).
        *   Trigger the Discord notification.

### Task 3.6: Implement Discord Webhook Sender
*   Create a function `sendDiscordNotification(webhookURL string, topTalkers map[string]float64)`.
*   Define Go structs matching the Discord embed structure.
*   Populate the embed struct with the top talkers data (IPs and their speeds in Mbps).
*   Marshal the struct to JSON.
*   Send a POST request using `net/http` to the `webhookURL` with the JSON payload.
*   Handle HTTP response and errors.

### Task 3.7: Implement Main Application Loop
*   In `main.go`:
    *   Load configuration.
    *   Initialize packet capture.
    *   Initialize traffic analysis components (ticker, aggregators).
    *   Start the packet reading goroutine.
    *   Start the analysis/ticker goroutine.
    *   Handle graceful shutdown (e.g., on SIGINT/SIGTERM) to close the packet capture handle.

### Task 3.8: Build Standalone Binary
*   Research `gopacket` CGO dependencies (`libpcap`).
*   Configure the build command for static linking:
    ```bash
    # Example for Linux
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -linkmode external -extldflags '-static'" -tags netgo -o network-monitor ./cmd/monitor
    ```
    *(Note: Exact flags might vary based on OS and target)*
*   Test the binary on a system without Go or `libpcap` installed.

### Task 3.9: Testing
*   Unit tests for configuration loading, speed calculation, Discord formatting.
*   Integration/manual testing:
    *   Run the monitor.
    *   Generate network traffic (e.g., using `iperf` or large downloads).
    *   Verify threshold triggering and Discord notifications.
    *   Test with different configurations.

### Task 3.10: Documentation (`README.md`)
*   Project description.
*   Prerequisites (if any, despite aiming for standalone).
*   Configuration options explanation.
*   Build instructions.
*   Usage instructions (including permission requirements).
*   Example Discord notification.

## 4. Considerations
*   **Permissions:** Packet capture typically requires elevated privileges. Document how users should run the application (e.g., `sudo` or capabilities `setcap cap_net_raw,cap_net_admin=eip network-monitor`).
*   **Performance:** High-speed networks might require optimization in packet processing. Consider sampling or potential bottlenecks.
*   **Cross-Compilation:** Building truly standalone binaries for different OS (Linux, macOS, Windows) requires specific static build setups for `libpcap`/`npcap` on each platform.
*   **IPv6 Support:** Ensure IPv6 source addresses are handled correctly if needed. 