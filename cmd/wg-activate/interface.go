package main

import (
	"net"
	"os"
	"syscall"
	"unsafe"
)

// FIXME: the next 3 functions are Linux only...
// we should check if the interface is pointtopoint before using

func parseAddr(ifam *syscall.IfAddrmsg, a syscall.NetlinkRouteAttr) *net.IPNet {
	if ifam.Family == syscall.AF_INET {
		return &net.IPNet{IP: net.IPv4(a.Value[0], a.Value[1], a.Value[2], a.Value[3]), Mask: net.CIDRMask(int(ifam.Prefixlen), 8*net.IPv4len)}
	} else if ifam.Family == syscall.AF_INET6 {
		ifa := &net.IPNet{IP: make(net.IP, net.IPv6len), Mask: net.CIDRMask(int(ifam.Prefixlen), 8*net.IPv6len)}
		copy(ifa.IP, a.Value[:])
		return ifa
	}
	return nil
}

func parsePointAddrs(ifam *syscall.IfAddrmsg, attrs []syscall.NetlinkRouteAttr) (localAddr *net.IPNet, peerAddr *net.IPNet) {
	for _, a := range attrs {
		if a.Attr.Type == syscall.IFA_LOCAL {
			localAddr = parseAddr(ifam, a)
		} else if a.Attr.Type == syscall.IFA_ADDRESS {
			peerAddr = parseAddr(ifam, a)
		}
	}
	// With IPv4, it seems that we can get IFA_LOCAL and IFA_ADDRESS
	// attributes that are identical. If that happens, reset localAddr to
	// nil and just return peerAddr
	if localAddr != nil && peerAddr != nil && localAddr.IP.Equal(peerAddr.IP) {
		localAddr = nil
	}
	return
}

func getInterfaceIPs(iface *net.Interface) (clientIPv4 *net.IPNet, clientIPv6 *net.IPNet, serverIPv4 *net.IPNet, serverIPv6 *net.IPNet, err error) {
	tab, err := syscall.NetlinkRIB(syscall.RTM_GETADDR, syscall.AF_UNSPEC)
	if err != nil {
		return clientIPv4, clientIPv6, serverIPv4, serverIPv6, os.NewSyscallError("netlinkrib", err)
	}
	msgs, err := syscall.ParseNetlinkMessage(tab)
	if err != nil {
		return clientIPv4, clientIPv6, serverIPv4, serverIPv6, os.NewSyscallError("parsenetlinkmessage", err)
	}

loop:
	for _, m := range msgs {
		switch m.Header.Type {
		case syscall.NLMSG_DONE:
			break loop
		case syscall.RTM_NEWADDR:
			ifam := (*syscall.IfAddrmsg)(unsafe.Pointer(&m.Data[0]))
			if iface.Index == int(ifam.Index) {
				attrs, err := syscall.ParseNetlinkRouteAttr(&m)
				if err != nil {
					return clientIPv4, clientIPv6, serverIPv4, serverIPv6, os.NewSyscallError("parsenetlinkrouteattr", err)
				}
				localAddr, peerAddr := parsePointAddrs(ifam, attrs)
				if localAddr != nil {
					if localAddr.IP.To4() != nil {
						clientIPv4 = peerAddr
						serverIPv4 = localAddr
					} else {
						clientIPv6 = peerAddr
						serverIPv6 = localAddr
					}
				} else {
					// There was no peer address returned, so increment server address by 1 to get client address
					clientAddr := &net.IPNet{IP: make(net.IP, len(peerAddr.IP)), Mask: make(net.IPMask, len(peerAddr.Mask))}
					copy(clientAddr.IP, peerAddr.IP)
					clientAddr.IP[len(clientAddr.IP)-1] += 1
					copy(clientAddr.Mask, peerAddr.Mask)

					// Check that the new client address is in the same network as the server
					// If not, reset to nil
					if !peerAddr.Contains(clientAddr.IP) {
						clientAddr = nil
					}

					if peerAddr.IP.To4() != nil {
						clientIPv4 = clientAddr
						serverIPv4 = peerAddr
					} else {
						clientIPv6 = clientAddr
						serverIPv6 = peerAddr
					}
				}
			}
		}
	}
	return
}
