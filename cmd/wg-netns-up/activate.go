package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	nsName     string
	clientIPv4 string
	serverIPv4 string
	clientIPv6 string
	serverIPv6 string
)

func init() {
	flag.StringVar(&nsName, "ns", "", "namespace name")
	flag.StringVar(&clientIPv4, "c4", "", "client IPv4 address (inside tunnel)")
	flag.StringVar(&serverIPv4, "s4", "", "server IPv4 address (inside tunnel)")
	flag.StringVar(&clientIPv6, "c6", "", "client IPv6 address (inside tunnel)")
	flag.StringVar(&serverIPv6, "s6", "", "server IPv6 address (inside tunnel)")
}

func main() {
	flag.Parse()

	if len(nsName) <= 0 {
		fmt.Fprintf(os.Stderr, "namespace name is required\n")
		os.Exit(1)
	}
	if len(clientIPv4) <= 0 {
		fmt.Fprintf(os.Stderr, "client IPv4 address (inside tunnel) is required\n")
		os.Exit(1)
	}
	if len(serverIPv4) <= 0 {
		fmt.Fprintf(os.Stderr, "server IPv4 address (inside tunnel) is required\n")
		os.Exit(1)
	}
	if len(clientIPv6) <= 0 {
		fmt.Fprintf(os.Stderr, "client IPv6 address (inside tunnel) is required\n")
		os.Exit(1)
	}
	if len(serverIPv6) <= 0 {
		fmt.Fprintf(os.Stderr, "server IPv6 address (inside tunnel) is required\n")
		os.Exit(1)
	}

	nsName = filepath.Base(nsName)
	nsPath := filepath.Join("/run/netns", nsName)

	if _, err := os.Stat(nsPath); err != nil {
		// network namespace does not already exist, create it

		cmd := exec.Command("ip", "netns", "add", nsName)
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error running command: %s\n", err)
			os.Exit(4)
		}

		cmd = exec.Command("ip", "link", "add", "dev", "wg0", "type", "wireguard")
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error running command: %s\n", err)
			os.Exit(4)
		}

		cmd = exec.Command("ip", "link", "set", "wg0", "netns", nsName)
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error running command: %s\n", err)
			os.Exit(4)
		}

		cmd = exec.Command("ip", "-n", nsName, "address", "add", "dev", "wg0", clientIPv4, "peer", serverIPv4)
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error running command: %s\n", err)
			os.Exit(4)
		}

		cmd = exec.Command("ip", "-n", nsName, "address", "add", "dev", "wg0", clientIPv6, "peer", serverIPv6)
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error running command: %s\n", err)
			os.Exit(4)
		}
	}

	cmd := exec.Command("ip", "netns", "exec", nsName, "wg", "setconf", "wg0", "/proc/self/fd/0")
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running command: %s\n", err)
		os.Exit(4)
	}

	cmd = exec.Command("ip", "-n", nsName, "link", "set", "wg0", "up")
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running command: %s\n", err)
		os.Exit(4)
	}

	cmd = exec.Command("ip", "-n", nsName, "route", "add", "default", "dev", "wg0")
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running command: %s\n", err)
		os.Exit(4)
	}

	cmd = exec.Command("ip", "-n", nsName, "-6", "route", "add", "default", "dev", "wg0")
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running command: %s\n", err)
		os.Exit(4)
	}
}
