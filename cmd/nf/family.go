package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
)

const familyUsage = `nf family — inspect the family roster

Usage:
  nf family list              List all loaded family members
  nf family show <name>       Show a single member as JSON
`

// familyMember mirrors the JSON shape returned by /api/v1/family*. Kept
// in the cli to avoid pulling in the server internals.
type familyMember struct {
	Name           string `json:"name"`
	Role           string `json:"role"`
	RiskTolerance  string `json:"risk_tolerance"`
	CostTier       string `json:"cost_tier"`
	MaxPRsPerNight int    `json:"max_prs_per_night"`
	Duties         []struct {
		Type     string `json:"type"`
		Interval string `json:"interval"`
	} `json:"duties"`
}

func familyCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, familyUsage)
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		familyList(args[1:])
	case "show":
		familyShow(args[1:])
	case "help", "-h", "--help":
		fmt.Print(familyUsage)
	default:
		fmt.Fprintf(os.Stderr, "nf family: unknown subcommand %q\n\n", args[0])
		fmt.Fprint(os.Stderr, familyUsage)
		os.Exit(2)
	}
}

func familyList(args []string) {
	fs := flag.NewFlagSet("family list", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "emit JSON instead of a table")
	_ = fs.Parse(args)

	var members []familyMember
	if err := apiGet("/api/v1/family", &members); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
		os.Exit(1)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(members)
		return
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tRISK\tCOST\tDUTIES\tROLE")
	for _, m := range members {
		duties := len(m.Duties)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\n",
			m.Name, m.RiskTolerance, m.CostTier, duties, m.Role)
	}
	_ = tw.Flush()
}

func familyShow(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "nf family show: <name> required")
		os.Exit(2)
	}
	name := args[0]
	var member map[string]any
	if err := apiGet("/api/v1/family/"+name, &member); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(member)
}
