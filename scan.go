package main

import (
	"net"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

func ScanIP(ip string) IPInfo {
	reachable := pingHost(ip)
	hostname := lookupHostname(ip)
	lastSeen := time.Time{}
	if reachable {
		lastSeen = time.Now()
	}
	return IPInfo{
		IP:        ip,
		Hostname:  hostname,
		Reachable: reachable,
		LastSeen:  lastSeen,
	}
}

func reverseLookup(ip string) string {
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return "---"
	}
	return names[0]
}

// Runs system ping command
func pingHost(ip string) bool {
	pinger, err := probing.NewPinger(ip)
	if err != nil {
		return false
	}

	pinger.Count = 3
	pinger.Timeout = 2 * time.Second
	pinger.SetPrivileged(false) // Required for most systems unless using setuid or capabilities

	err = pinger.Run() // Blocks until done
	if err != nil {
		return false
	}

	stats := pinger.Statistics()
	return stats.PacketsRecv > 0
}
