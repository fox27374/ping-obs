package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	htmlFile = "index.html"
)

var (
	ipList    []string
	statusMap = make(map[string]*hostDetails)
	mu        sync.RWMutex
)

type hostDetails struct {
	IP       string
	Hostname string
	Status   bool
	LastSeen time.Time
}

func main() {
	webMode := flag.Bool("web", false, "Run web server with live status")
	flag.Parse()

	ipsArgs := flag.Args()
	if len(ipsArgs) == 0 {
		fmt.Println("Usage: go run main.go [--web] <ip/range/network> [<ip/range/network> ...]")
		os.Exit(1)
	}

	// Parse IP inputs
	for _, arg := range ipsArgs {
		ips, err := parseInput(arg)
		if err != nil {
			fmt.Printf("Error parsing %s: %v\n", arg, err)
			os.Exit(1)
		}
		ipList = append(ipList, ips...)
	}

	mu.Lock()
	for _, ip := range ipList {
		statusMap[ip] = &hostDetails{
			IP:       ip,
			Hostname: "---",
			Status:   false,
			LastSeen: time.Time{},
		}
	}
	mu.Unlock()

	go func() {
		for {
			checkAllIPs()
			time.Sleep(5 * time.Second)
		}
	}()

	if *webMode {
		startWebServer()
	} else {
		// CLI mode: print status every 5 seconds
		for {
			printStatusTable(ipList, statusMap)
			time.Sleep(5 * time.Second)
		}
	}
}

func checkAllIPs() {
	var wg sync.WaitGroup
	for _, ip := range ipList {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			pingable := pingHost(ip)
			hostname := "---"
			if pingable {
				names, err := net.LookupAddr(ip)
				if err == nil && len(names) > 0 {
					hostname = strings.TrimSuffix(names[0], ".")
				}
			}

			mu.Lock()
			info := statusMap[ip]
			info.Status = pingable
			info.Hostname = hostname
			if pingable {
				info.LastSeen = time.Now()
			}
			mu.Unlock()
		}(ip)
	}
	wg.Wait()
}
