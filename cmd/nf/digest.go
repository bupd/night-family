package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const digestUsage = `nf digest — read a night's morning markdown digest

Usage:
  nf digest show <night-id>     Print the digest for a night
`

func digestCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, digestUsage)
		os.Exit(2)
	}
	switch args[0] {
	case "show":
		digestShow(args[1:])
	case "help", "-h", "--help":
		fmt.Print(digestUsage)
	default:
		fmt.Fprintf(os.Stderr, "nf digest: unknown subcommand %q\n\n", args[0])
		fmt.Fprint(os.Stderr, digestUsage)
		os.Exit(2)
	}
}

func digestShow(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "nf digest show: <night-id> required")
		os.Exit(2)
	}
	c := &http.Client{Timeout: 10 * time.Second}
	resp, err := c.Get(daemonURL() + "/api/v1/nights/" + args[0] + "/digest")
	if err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		fmt.Fprintf(os.Stderr, "nf: daemon returned %s: %s\n", resp.Status, string(b))
		os.Exit(1)
	}
	_, _ = io.Copy(os.Stdout, resp.Body)
}
