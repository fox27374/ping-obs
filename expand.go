package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func ExpandInput(input string) ([]string, error) {
	switch {
	case strings.Contains(input, "/"):
		return ExpandCIDR(input)
	case strings.Contains(input, "-"):
		return ExpandIPRange(input)
	default:
		return ExpandSingleIP(input)
	}
}

func ExpandCIDR(cidr string) ([]string, error) {
	var ips []string
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}
	if len(ips) > 2 {
		ips = ips[1 : len(ips)-1] // remove network and broadcast
	}
	return ips, nil
}

func ExpandIPRange(rng string) ([]string, error) {
	parts := strings.Split(rng, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid range format")
	}

	baseIP := net.ParseIP(parts[0]).To4()
	if baseIP == nil {
		return nil, fmt.Errorf("invalid IP: %s", parts[0])
	}

	start := int(baseIP[3])
	end, err := strconv.Atoi(parts[1])
	if err != nil || end < start || end > 255 {
		return nil, fmt.Errorf("invalid end range: %s", parts[1])
	}

	var ips []string
	for i := start; i <= end; i++ {
		ip := net.IPv4(baseIP[0], baseIP[1], baseIP[2], byte(i))
		ips = append(ips, ip.String())
	}
	return ips, nil
}

func ExpandSingleIP(ip string) ([]string, error) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return nil, fmt.Errorf("invalid IP: %s", ip)
	}
	return []string{parsed.String()}, nil
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
