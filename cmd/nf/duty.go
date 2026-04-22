package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
)

const dutyUsage = `nf duty — inspect the duty catalogue

Usage:
  nf duty list              List known duty types
  nf duty show <type>       Show metadata for one duty type
`

type dutyInfo struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Output      string `json:"output"`
	CostTier    string `json:"cost_tier"`
	Risk        string `json:"risk"`
	Builtin     bool   `json:"builtin"`
}

func dutyCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, dutyUsage)
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		dutyList(args[1:])
	case "show":
		dutyShow(args[1:])
	case "help", "-h", "--help":
		fmt.Print(dutyUsage)
	default:
		fmt.Fprintf(os.Stderr, "nf duty: unknown subcommand %q\n\n", args[0])
		fmt.Fprint(os.Stderr, dutyUsage)
		os.Exit(2)
	}
}

func dutyList(args []string) {
	fs := flag.NewFlagSet("duty list", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "emit JSON instead of a table")
	_ = fs.Parse(args)

	var duties []dutyInfo
	if err := apiGet("/api/v1/duties", &duties); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
		os.Exit(1)
	}
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(duties)
		return
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TYPE\tOUTPUT\tCOST\tRISK\tDESCRIPTION")
	for _, d := range duties {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			d.Type, d.Output, d.CostTier, d.Risk, d.Description)
	}
	if err := tw.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
	}
}

func dutyShow(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "nf duty show: <type> required")
		os.Exit(2)
	}
	var info map[string]any
	if err := apiGet("/api/v1/duties/"+args[0], &info); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(info)
}
