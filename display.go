package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func startWebServer() {
	pageHTML := getHtmlContent(htmlFile)
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

func getHtmlContent(c string) string {
	htmlBytes, err := os.ReadFile(c)
	if err != nil {
		log.Fatalf("Failed to read HTML file: %v", err)
	}
	htmlContent := string(htmlBytes)
	return htmlContent
}

func printStatusTable(allIPs []string, ipStatus map[string]*hostDetails) {
	fmt.Print("\033[H\033[2J")
	fmt.Printf("%-15s %-30s %-10s %s\n", "IP", "Hostname", "Status", "Last Seen")
	fmt.Println(strings.Repeat("-", 70))
	for _, ip := range allIPs {
		info := ipStatus[ip]
		var seen string
		if info.Status && !info.LastSeen.IsZero() {
			ago := time.Since(info.LastSeen).Round(time.Second)
			seen = fmt.Sprintf("%vs ago", int(ago.Seconds()))
		} else {
			seen = "never"
		}
		statusUpDown := "DOWN"
		if info.Status {
			statusUpDown = "UP"
		}

		fmt.Printf("%-15s %-30s %-10v %s\n", info.IP, info.Hostname, statusUpDown, seen)
	}
}
