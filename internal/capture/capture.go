package capture

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

const (
	snapshotLen int32         = 1024
	promiscuous bool          = true
	timeout     time.Duration = -1 * time.Second

	bpfFilter string = "ip or ip6"
)

func StartCapture(interfaceName string) (*gopacket.PacketSource, *pcap.Handle, error) {
	var handle *pcap.Handle
	var err error

	if interfaceName == "" {

		devices, err := pcap.FindAllDevs()
		if err != nil {
			return nil, nil, fmt.Errorf("error finding devices: %w", err)
		}

		if len(devices) == 0 {
			return nil, nil, errors.New("no network interfaces found")
		}

		for _, device := range devices {

			if strings.HasPrefix(device.Name, "lo") {
				continue
			}

			if len(device.Addresses) == 0 {
				continue
			}
			log.Printf("No interface specified, using first valid device found: %s", device.Name)
			interfaceName = device.Name
			break
		}
		if interfaceName == "" {
			return nil, nil, errors.New("no suitable network interface found (non-loopback with addresses)")
		}
	}

	handle, err = pcap.OpenLive(interfaceName, snapshotLen, promiscuous, timeout)
	if err != nil {

		if strings.Contains(strings.ToLower(err.Error()), "permission denied") {
			return nil, nil, fmt.Errorf("permission denied opening interface %s. Run with sudo or set capabilities (e.g., sudo setcap cap_net_raw,cap_net_admin=eip <your_binary>)", interfaceName)
		}
		return nil, nil, fmt.Errorf("error opening device %s: %w", interfaceName, err)
	}

	log.Printf("Using BPF filter: %s", bpfFilter)
	err = handle.SetBPFFilter(bpfFilter)
	if err != nil {
		handle.Close()
		return nil, nil, fmt.Errorf("error setting BPF filter '%s': %w", bpfFilter, err)
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	log.Printf("Successfully opened interface %s for capture.", interfaceName)

	return packetSource, handle, nil
}
