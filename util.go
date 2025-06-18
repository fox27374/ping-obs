package main

import (
	"net"
	"sort"
)

func sortIPs(ips []string) []string {
	sort.Slice(ips, func(i, j int) bool {
		ip1 := net.ParseIP(ips[i]).To4()
		ip2 := net.ParseIP(ips[j]).To4()
		if ip1 == nil || ip2 == nil {
			return ips[i] < ips[j]
		}
		return bytesCompare(ip1, ip2) < 0
	})
	return ips
}

func bytesCompare(a, b net.IP) int {
	for i := 0; i < len(a); i++ {
		if a[i] < b[i] {
			return -1
		} else if a[i] > b[i] {
			return 1
		}
	}
	return 0
}
