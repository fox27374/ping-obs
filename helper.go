package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

func parseInput(input string) ([]string, error) {
	// Parse CIDR network
	if strings.Contains(input, "/") {
		_, ipnet, err := net.ParseCIDR(input)
		if err != nil {
			return nil, err
		}
		return ipsFromCIDR(ipnet), nil

		// Parse network range
	} else if strings.Contains(input, "-") {
		return ipsFromRange(input)

		// Parse single IP
	} else {
		ip := net.ParseIP(input)
		if ip == nil {
			return nil, fmt.Errorf("invalid IP %s", input)
		}
		return []string{ip.String()}, nil
	}
}

func isBroadcast(ip net.IP, ipnet *net.IPNet) bool {
	broadcast := make(net.IP, len(ipnet.IP))
	for i := range ipnet.IP {
		broadcast[i] = ipnet.IP[i] | ^ipnet.Mask[i]
	}
	return ip.Equal(broadcast)
}

func ipsFromCIDR(ipnet *net.IPNet) []string {
	var ips []string
	for ip := ipnet.IP.Mask(ipnet.Mask); ipnet.Contains(ip); ip = incIP(ip) {
		if !ip.Equal(ipnet.IP) && !isBroadcast(ip, ipnet) {
			ips = append(ips, ip.String())
		}
	}
	return ips
}

func ipsFromRange(r string) ([]string, error) {
	// Example: 192.168.1.1-10
	parts := strings.Split(r, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid range format")
	}
	startIP := net.ParseIP(parts[0])
	if startIP == nil {
		return nil, fmt.Errorf("invalid start IP")
	}
	startIP4 := startIP.To4()
	if startIP4 == nil {
		return nil, fmt.Errorf("only IPv4 supported for ranges")
	}
	prefix := strings.Join(strings.Split(parts[0], ".")[:3], ".") + "."
	startLastOctet := startIP4[3]
	var endLastOctet int
	_, err := fmt.Sscanf(parts[1], "%d", &endLastOctet)
	if err != nil {
		return nil, fmt.Errorf("invalid range end")
	}
	if endLastOctet < int(startLastOctet) || endLastOctet > 255 {
		return nil, fmt.Errorf("invalid range end")
	}
	var ips []string
	for i := int(startLastOctet); i <= endLastOctet; i++ {
		ips = append(ips, fmt.Sprintf("%s%d", prefix, i))
	}
	return ips, nil
}

func incIP(ip net.IP) net.IP {
	ip = ip.To4()
	if ip == nil {
		return nil
	}
	out := make(net.IP, len(ip))
	copy(out, ip)
	for i := len(out) - 1; i >= 0; i-- {
		out[i]++
		if out[i] > 0 {
			break
		}
	}
	return out
}

func pingHost(ip string) bool {
	pinger, err := probing.NewPinger(ip)
	if err != nil {
		return false
	}

	pinger.Count = 3
	pinger.Timeout = 2 * time.Second
	pinger.SetPrivileged(false) // Set to unprivileged, otherwise root/admin rights are needed

	err = pinger.Run() // Blocks until done
	if err != nil {
		return false
	}

	stats := pinger.Statistics()
	return stats.PacketsRecv > 0
}
