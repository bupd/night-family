package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// scheduleCmd maps to "nf schedule show".
func scheduleCmd(args []string) {
	if len(args) > 0 {
		switch args[0] {
		case "help", "-h", "--help":
			fmt.Println("nf schedule show    Print the current window + next fire")
			return
		case "show":
		default:
			fmt.Fprintln(os.Stderr, "nf schedule: unknown subcommand")
			os.Exit(2)
		}
	}
	var body map[string]any
	if err := apiGet("/api/v1/schedule", &body); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(body)
}

// budgetCmd maps to "nf budget show".
func budgetCmd(args []string) {
	if len(args) > 0 && args[0] != "show" {
		fmt.Fprintln(os.Stderr, "nf budget: usage: nf budget show")
		os.Exit(2)
	}
	var body map[string]any
	if err := apiGet("/api/v1/budget", &body); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(body)
}

// statsCmd maps to "nf stats".
func statsCmd(_ []string) {
	var body map[string]any
	if err := apiGet("/api/v1/stats", &body); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(body)
}

// doctorCmd probes the daemon for basic liveness + prints whatever it
// knows about the running config. A quick `can-it-even-run` diagnostic.
func doctorCmd(_ []string) {
	reports := []struct {
		name string
		path string
	}{
		{"daemon reachable", "/healthz"},
		{"daemon ready", "/readyz"},
		{"version", "/version"},
		{"schedule configured", "/api/v1/schedule"},
		{"stats available", "/api/v1/stats"},
	}
	client := &http.Client{Timeout: 5 * time.Second}
	ok := true
	for _, r := range reports {
		resp, err := client.Get(daemonURL() + r.path)
		if err != nil {
			fmt.Printf("✗ %-25s %s: %v\n", r.name, r.path, err)
			ok = false
			continue
		}
		resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			fmt.Printf("✗ %-25s %s → %s\n", r.name, r.path, resp.Status)
			ok = false
			continue
		}
		fmt.Printf("✓ %-25s %s → %s\n", r.name, r.path, resp.Status)
	}
	if !ok {
		os.Exit(1)
	}
}
