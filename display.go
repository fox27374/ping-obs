package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func printStatusTable(allIPs []string, ipStatus map[string]IPInfo) {
	fmt.Print("\033[H\033[2J")
	fmt.Printf("%-15s %-30s %-10s %s\n", "IP", "Hostname", "Status", "Last Seen")
	fmt.Println(strings.Repeat("-", 70))
	for _, ip := range allIPs {
		info := ipStatus[ip]
		var seen string
		if info.Reachable && !info.LastSeen.IsZero() {
			ago := time.Since(info.LastSeen).Round(time.Second)
			seen = fmt.Sprintf("%vs ago", int(ago.Seconds()))
		} else {
			seen = "never"
		}
		fmt.Printf("%-15s %-30s %-10v %s\n", info.IP, info.Hostname, info.Reachable, seen)
	}
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

var pageHTML = `
	<!DOCTYPE html>
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
	</html>
`
