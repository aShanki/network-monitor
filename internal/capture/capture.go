package capture

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

const (
	snapshotLen int32         = 1024
	promiscuous bool          = true
	timeout     time.Duration = -1 * time.Second // Negative timeout means wait indefinitely
	// Simple BPF filter to capture only IPv4 and IPv6 traffic.
	// Adjust as needed, e.g., "tcp port 80"
	bpfFilter string = "ip or ip6"
)

// StartCapture opens the specified network interface or finds the first available one
// if interfaceName is empty. It applies a BPF filter and returns a packet source.
func StartCapture(interfaceName string) (*gopacket.PacketSource, *pcap.Handle, error) {
	var handle *pcap.Handle
	var err error

	if interfaceName == "" {
		// Find all devices
		devices, err := pcap.FindAllDevs()
		if err != nil {
			return nil, nil, fmt.Errorf("error finding devices: %w", err)
		}

		if len(devices) == 0 {
			return nil, nil, errors.New("no network interfaces found")
		}

		// Use the first available device that's not loopback
		for _, device := range devices {
			// Skip loopback interfaces
			if (device.Flags & pcap.FlagLoopback) == pcap.FlagLoopback {
				continue
			}
			// Skip interfaces without IP addresses (often virtual)
			if len(device.Addresses) == 0 {
				continue
			}
			log.Printf("No interface specified, using first valid device found: %s", device.Name)
			interfaceName = device.Name
			break // Use the first non-loopback interface with an address
		}
		if interfaceName == "" {
			return nil, nil, errors.New("no suitable network interface found (non-loopback with addresses)")
		}
	}

	// Open device
	handle, err = pcap.OpenLive(interfaceName, snapshotLen, promiscuous, timeout)
	if err != nil {
		// Common error on Linux without sufficient privileges
		if errors.Is(err, pcap.ErrPermissionDenied) {
			return nil, nil, fmt.Errorf("permission denied opening interface %s. Run with sudo or set capabilities (e.g., sudo setcap cap_net_raw,cap_net_admin=eip <your_binary>)", interfaceName)
		}
		return nil, nil, fmt.Errorf("error opening device %s: %w", interfaceName, err)
	}

	// Set BPF filter
	log.Printf("Using BPF filter: %s", bpfFilter)
	err = handle.SetBPFFilter(bpfFilter)
	if err != nil {
		handle.Close() // Close handle on error
		return nil, nil, fmt.Errorf("error setting BPF filter '%s': %w", bpfFilter, err)
	}

	// Use the handle as a packet source
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	log.Printf("Successfully opened interface %s for capture.", interfaceName)

	return packetSource, handle, nil
}
