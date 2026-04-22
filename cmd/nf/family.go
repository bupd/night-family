package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

// familyWrite is shared by add / replace / validate. It reads a YAML
// file, converts to JSON, and posts to the daemon. When usePath is
// true the path is used as-is; otherwise for replace we build it from
// the member's name.
func familyWrite(args []string, method, path string, usePath bool) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "nf family: <file.yaml> required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
		os.Exit(1)
	}
	var parsed map[string]any
	if err := yaml.Unmarshal(raw, &parsed); err != nil {
		fmt.Fprintln(os.Stderr, "nf: invalid YAML:", err)
		os.Exit(1)
	}
	body, _ := json.Marshal(parsed)

	if !usePath {
		name, _ := parsed["name"].(string)
		if method == "PUT" {
			if name == "" {
				fmt.Fprintln(os.Stderr, "nf: missing name field; PUT target unknown")
				os.Exit(1)
			}
			path = "/api/v1/family/" + name
		}
	}

	var resp map[string]any
	var herr error
	switch method {
	case "POST":
		herr = apiPost(path, body, &resp)
	case "PUT":
		herr = apiJSON("PUT", path, body, &resp)
	}
	if herr != nil {
		fmt.Fprintln(os.Stderr, "nf:", herr)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(resp)
}

func familyRemove(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "nf family remove: <name> required")
		os.Exit(2)
	}
	if err := apiJSON("DELETE", "/api/v1/family/"+args[0], nil, nil); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
		os.Exit(1)
	}
	fmt.Printf("removed: %s\n", args[0])
}

const familyUsage = `nf family — inspect or mutate the family roster

Usage:
  nf family list                  List all loaded family members
  nf family show <name>           Show a single member as JSON
  nf family add <file.yaml>       POST a new member (conflict = 409)
  nf family replace <file.yaml>   PUT replace an existing member
  nf family remove <name>         DELETE a member
  nf family validate <file.yaml>  Check a file without storing
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
	case "add":
		familyWrite(args[1:], "POST", "/api/v1/family", true)
	case "replace":
		familyWrite(args[1:], "PUT", "", false)
	case "remove":
		familyRemove(args[1:])
	case "validate":
		familyWrite(args[1:], "POST", "/api/v1/family/validate", false)
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
	if err := tw.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
	}
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
