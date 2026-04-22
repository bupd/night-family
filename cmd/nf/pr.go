package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
)

const prUsage = `nf pr — inspect PRs opened by the family

Usage:
  nf pr list [--json]
`

func prCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, prUsage)
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		prList(args[1:])
	case "help", "-h", "--help":
		fmt.Print(prUsage)
	default:
		fmt.Fprintf(os.Stderr, "nf pr: unknown subcommand %q\n\n", args[0])
		fmt.Fprint(os.Stderr, prUsage)
		os.Exit(2)
	}
}

func prList(args []string) {
	fs := flag.NewFlagSet("pr list", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "emit JSON instead of a table")
	_ = fs.Parse(args)
	var body map[string]any
	if err := apiGet("/api/v1/prs", &body); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
		os.Exit(1)
	}
	items, _ := body["items"].([]any)
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(items)
		return
	}
	if len(items) == 0 {
		fmt.Println("(no PRs)")
		return
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "OPENED\tMEMBER\tDUTY\tSTATE\tURL")
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			m["opened_at"], m["member"], m["duty"], m["state"], m["url"])
	}
	if err := tw.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
	}
}
