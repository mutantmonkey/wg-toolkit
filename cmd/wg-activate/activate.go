package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type ClientConfigBlock struct {
	PublicKey    string `json:"PublicKey"`
	PresharedKey string `json:"PresharedKey,omitempty"`
	MTU          uint32 `json:"MTU,omitempty"`
}

type ClientConfigResponse struct {
	Port       int    `json:"Port"`
	PublicKey  string `json:"PublicKey"`
	ClientIPv4 string `json:"ClientIPv4,omitempty"`
	ClientIPv6 string `json:"ClientIPv6,omitempty"`
	ServerIPv4 string `json:"ServerIPv4,omitempty"`
	ServerIPv6 string `json:"ServerIPv6,omitempty"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: wg-activate <device name>\n")
		os.Exit(1)
	}

	clientConfig := &ClientConfigBlock{}
	inputBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Client config JSON must be provided on stdin.\n")
		os.Exit(2)
	}

	err = json.Unmarshal(inputBytes, clientConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing client config JSON.\n")
		os.Exit(3)
	}

	client, err := wgctrl.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create WireGuard client: %s\n", err)
		os.Exit(4)
	}

	name := os.Args[1]
	dev, err := client.Device(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to access interface: %s\n", err)
		os.Exit(5)
	}

	iface, err := net.InterfaceByName(name)
	if err != nil {
		panic(err)
	}

	config := wgtypes.Config{
		ReplacePeers: true,
	}

	clientPublicKey, err := wgtypes.ParseKey(clientConfig.PublicKey)
	if err != nil {
		panic(err)
	}

	peerConfig := wgtypes.PeerConfig{
		PublicKey:         clientPublicKey,
		ReplaceAllowedIPs: true,
	}

	if clientConfig.PresharedKey != "" {
		clientPresharedKey, err := wgtypes.ParseKey(clientConfig.PresharedKey)
		if err != nil {
			panic(err)
		}
		peerConfig.PresharedKey = &clientPresharedKey
	}

	resp := ClientConfigResponse{}

	clientIPv4, clientIPv6, serverIPv4, serverIPv6, err := getInterfaceIPs(iface)
	if err != nil {
		panic(err)
	}
	if clientIPv4 != nil {
		peerConfig.AllowedIPs = append(peerConfig.AllowedIPs, *clientIPv4)
		resp.ClientIPv4 = clientIPv4.String()
	}
	if clientIPv6 != nil {
		peerConfig.AllowedIPs = append(peerConfig.AllowedIPs, *clientIPv6)
		resp.ClientIPv6 = clientIPv6.String()
	}
	if serverIPv4 != nil {
		resp.ServerIPv4 = serverIPv4.String()
	}
	if serverIPv6 != nil {
		resp.ServerIPv6 = serverIPv6.String()
	}

	config.Peers = append(config.Peers, peerConfig)

	// reconfigure device to actually set up the peer
	err = client.ConfigureDevice(name, config)
	if err != nil {
		panic(err)
	}

	resp.Port = dev.ListenPort
	resp.PublicKey = dev.PublicKey.String()

	b, err := json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(b)
	os.Stdout.Write([]byte("\n"))
}
