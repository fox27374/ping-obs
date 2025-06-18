package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

type IPInfo struct {
	IP        string
	Hostname  string
	Reachable bool
	LastSeen  time.Time
}

var (
	ipList    []string
	statusMap = make(map[string]*IPInfo)
	mu        sync.RWMutex
)

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
		statusMap[ip] = &IPInfo{
			IP:        ip,
			Hostname:  "---",
			Reachable: false,
			LastSeen:  time.Time{},
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
			printStatus()
			time.Sleep(5 * time.Second)
		}
	}
}

func parseInput(input string) ([]string, error) {
	if strings.Contains(input, "/") {
		_, ipnet, err := net.ParseCIDR(input)
		if err != nil {
			return nil, err
		}
		return ipsFromCIDR(ipnet), nil
	} else if strings.Contains(input, "-") {
		return ipsFromRange(input)
	} else {
		ip := net.ParseIP(input)
		if ip == nil {
			return nil, fmt.Errorf("invalid IP %s", input)
		}
		return []string{ip.String()}, nil
	}
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

func isBroadcast(ip net.IP, ipnet *net.IPNet) bool {
	broadcast := make(net.IP, len(ipnet.IP))
	for i := range ipnet.IP {
		broadcast[i] = ipnet.IP[i] | ^ipnet.Mask[i]
	}
	return ip.Equal(broadcast)
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
			info.Reachable = pingable
			info.Hostname = hostname
			if pingable {
				info.LastSeen = time.Now()
			}
			mu.Unlock()
		}(ip)
	}
	wg.Wait()
}

func pingHost(ip string) bool {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "-n", "1", "-w", "1000", ip)
	} else {
		cmd = exec.Command("ping", "-c", "1", "-W", "1", ip)
	}
	err := cmd.Run()
	return err == nil
}

func printStatus() {
	mu.RLock()
	defer mu.RUnlock()
	fmt.Println("IP\t\tHostname\tReachable\tLast Seen (s ago)")
	for _, ip := range ipList {
		info := statusMap[ip]
		lastSeenStr := "never"
		if info.Reachable && !info.LastSeen.IsZero() {
			lastSeenStr = fmt.Sprintf("%d", int(time.Since(info.LastSeen).Seconds()))
		}
		fmt.Printf("%s\t%s\t%v\t%s\n", info.IP, info.Hostname, info.Reachable, lastSeenStr)
	}
	fmt.Println()
}

func startWebServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, pageHTML)
	})

	http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		b, _ := json.Marshal(statusMap)
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})

	fmt.Println("Server running at http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

var pageHTML = `<!DOCTYPE html>
<html>
<head>
  <title>IP Scanner</title>
  <style>
    body { font-family: sans-serif; padding: 2em; }
    table { border-collapse: collapse; width: 100%; }
    th, td { border: 1px solid #ccc; padding: 8px; text-align: left; cursor: pointer; }
    th { background: #f4f4f4; user-select: none; }
    th.sort-asc::after {
      content: " ▲";
    }
    th.sort-desc::after {
      content: " ▼";
    }
    input {
      margin-bottom: 1em;
      padding: 0.3em 0.6em;
      font-size: 1em;
      width: 100%;
      max-width: 300px;
    }
  </style>
</head>
<body>
  <h1>IP Scanner</h1>

  <label for="filterInput">Filter IP / Hostname:</label><br>
  <input type="text" id="filterInput" placeholder="Type to filter...">

  <table id="ipTable">
    <thead>
      <tr>
        <th data-col="IP">IP</th>
        <th data-col="Hostname">Hostname</th>
        <th data-col="Reachable">Reachable</th>
        <th data-col="LastSeen">Last Seen</th>
      </tr>
    </thead>
    <tbody></tbody>
  </table>

  <script>
    let sortColumn = 'IP';
    let sortAsc = true;
    let currentData = {};

    async function fetchData() {
      const res = await fetch('/api/status');
      currentData = await res.json();
      renderTable();
    }

    function renderTable() {
      const tbody = document.querySelector('#ipTable tbody');
      const filterText = document.getElementById('filterInput').value.toLowerCase();

      let keys = Object.keys(currentData).filter(ip => {
        const info = currentData[ip];
        return ip.toLowerCase().includes(filterText) || info.Hostname.toLowerCase().includes(filterText);
      });

      keys.sort((a, b) => {
        const aInfo = currentData[a];
        const bInfo = currentData[b];
        switch (sortColumn) {
          case 'IP':
            return sortAsc ? ipToInt(a) - ipToInt(b) : ipToInt(b) - ipToInt(a);
          case 'Hostname':
            return sortAsc
              ? aInfo.Hostname.localeCompare(bInfo.Hostname)
              : bInfo.Hostname.localeCompare(aInfo.Hostname);
          case 'Reachable':
            return sortAsc
              ? (aInfo.Reachable === bInfo.Reachable ? 0 : aInfo.Reachable ? -1 : 1)
              : (aInfo.Reachable === bInfo.Reachable ? 0 : aInfo.Reachable ? 1 : -1);
          case 'LastSeen':
            const aTime = aInfo.LastSeen ? new Date(aInfo.LastSeen).getTime() : 0;
            const bTime = bInfo.LastSeen ? new Date(bInfo.LastSeen).getTime() : 0;
            return sortAsc ? aTime - bTime : bTime - aTime;
        }
        return 0;
      });

      tbody.innerHTML = '';
      keys.forEach(ip => {
        const row = currentData[ip];
        let seen = "never";
        if (row.Reachable && new Date(row.LastSeen).getFullYear() > 2000) {
          seen = timeAgo(row.LastSeen);
        }
        const bgColor = row.Reachable ? '#d4edda' : '#f8d7da';
        tbody.innerHTML += "<tr style=\"background-color: " + bgColor + ";\">" +
          "<td>" + row.IP + "</td>" +
          "<td>" + row.Hostname + "</td>" +
          "<td>" + row.Reachable + "</td>" +
          "<td>" + seen + "</td>" +
          "</tr>";
      });
    }

    function ipToInt(ip) {
      return ip.split('.').reduce((acc, octet) => (acc << 8) + parseInt(octet), 0);
    }

    function timeAgo(dateStr) {
      const seconds = Math.floor((Date.now() - new Date(dateStr)) / 1000);
      if (seconds < 60) return seconds + "s ago";
      const minutes = Math.floor(seconds / 60);
      if (minutes < 60) return minutes + "m ago";
      const hours = Math.floor(minutes / 60);
      if (hours < 24) return hours + "h ago";
      const days = Math.floor(hours / 24);
      return days + "d ago";
    }

    document.querySelectorAll('th').forEach(th => {
      th.addEventListener('click', () => {
        const col = th.getAttribute('data-col');
        if (col === sortColumn) {
          sortAsc = !sortAsc;
        } else {
          sortColumn = col;
          sortAsc = true;
        }
        document.querySelectorAll('th').forEach(th2 => th2.classList.remove('sort-asc', 'sort-desc'));
        th.classList.add(sortAsc ? 'sort-asc' : 'sort-desc');
        renderTable();
      });
    });

    document.getElementById('filterInput').addEventListener('input', renderTable);

    fetchData();
    setInterval(fetchData, 5000);
  </script>
</body>
</html>`
