package main

import (
	"fmt"
	"os"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: wg-prune <device name> <cutoff time>\n")
		os.Exit(1)
	}

	client, err := wgctrl.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create WireGuard client: %s\n", err)
		os.Exit(2)
	}

	name := os.Args[1]
	dev, err := client.Device(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to access interface: %s\n", err)
		os.Exit(3)
	}

	cutoffDuration, err := time.ParseDuration(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse duration: %s\n", err)
		os.Exit(4)
	}

	config := wgtypes.Config{}

	for _, peer := range dev.Peers {
		// if last handshake time is not zero and the last handshake
		// occurred more than cutoffDuration ago, remove the peer
		if !peer.LastHandshakeTime.IsZero() && peer.LastHandshakeTime.Add(cutoffDuration).Before(time.Now()) {
			config.Peers = append(config.Peers, wgtypes.PeerConfig{
				PublicKey: peer.PublicKey,
				Remove:    true,
			})
		}
	}

	// reconfigure device to actually remove the peers
	err = client.ConfigureDevice(name, config)
	if err != nil {
		panic(err)
	}
}
