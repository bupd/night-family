package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
)

func nightTrigger(args []string) {
	fs := flag.NewFlagSet("night trigger", flag.ExitOnError)
	members := fs.String("only-members", "", "comma-separated member names to include (default: all)")
	duties := fs.String("only-duties", "", "comma-separated duty types to include (default: all)")
	budget := fs.Int("budget", 0, "token budget ceiling (0 = unlimited)")
	dryRun := fs.Bool("dry-run", false, "plan the night but do not dispatch runs")
	_ = fs.Parse(args)

	body := map[string]any{}
	if *members != "" {
		body["only_members"] = strings.Split(*members, ",")
	}
	if *duties != "" {
		body["only_duties"] = strings.Split(*duties, ",")
	}
	if *budget > 0 {
		body["budget"] = *budget
	}
	if *dryRun {
		body["dry_run"] = true
	}
	raw, _ := json.Marshal(body)
	var res map[string]any
	if err := apiPost("/api/v1/nights/trigger", raw, &res); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(res)
}

func nightList(_ []string) {
	var body map[string]any
	if err := apiGet("/api/v1/nights", &body); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
		os.Exit(1)
	}
	items, _ := body["items"].([]any)
	if len(items) == 0 {
		fmt.Println("(no nights)")
		return
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSTARTED\tFINISHED\tSUMMARY")
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		started, _ := m["started_at"].(string)
		finished, _ := m["finished_at"].(string)
		if finished == "" {
			finished = "—"
		}
		summary, _ := m["summary"].(string)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", m["id"], started, finished, summary)
	}
	if err := tw.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
	}
}

func nightShow(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "nf night show: <id> required")
		os.Exit(2)
	}
	var body map[string]any
	if err := apiGet("/api/v1/nights/"+args[0], &body); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(body)
}

const nightUsage = `nf night — inspect or trigger nightly plans

Usage:
  nf night preview [--budget N] [--json]                          Show what would run if a night started now
  nf night trigger [--only-members X,Y] [--only-duties A,B] [--dry-run] [--budget N]
                                                                  Dispatch a night synchronously via the daemon's provider
  nf night list                                                   List past nights
  nf night show <id>                                              Show one night
`

type nightSlot struct {
	Member          string `json:"member"`
	Duty            string `json:"duty"`
	Priority        string `json:"priority"`
	CostTier        string `json:"cost_tier"`
	Risk            string `json:"risk"`
	Output          string `json:"output"`
	EstimatedTokens int    `json:"estimated_tokens"`
	Reason          string `json:"reason,omitempty"`
}

type nightSkipped struct {
	Member string `json:"member"`
	Duty   string `json:"duty"`
	Reason string `json:"reason"`
}

type nightPlan struct {
	WindowStart    string         `json:"window_start"`
	WindowEnd      string         `json:"window_end"`
	BudgetTokens   int            `json:"budget_tokens"`
	ReservedTokens int            `json:"reserved_tokens"`
	Slots          []nightSlot    `json:"slots"`
	Skipped        []nightSkipped `json:"skipped"`
}

func nightCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, nightUsage)
		os.Exit(2)
	}
	switch args[0] {
	case "preview":
		nightPreview(args[1:])
	case "trigger":
		nightTrigger(args[1:])
	case "list":
		nightList(args[1:])
	case "show":
		nightShow(args[1:])
	case "help", "-h", "--help":
		fmt.Print(nightUsage)
	default:
		fmt.Fprintf(os.Stderr, "nf night: unknown subcommand %q\n\n", args[0])
		fmt.Fprint(os.Stderr, nightUsage)
		os.Exit(2)
	}
}

func nightPreview(args []string) {
	fs := flag.NewFlagSet("night preview", flag.ExitOnError)
	budget := fs.Int("budget", 0, "token budget ceiling (0 = unlimited)")
	jsonOut := fs.Bool("json", false, "emit JSON instead of a table")
	_ = fs.Parse(args)

	path := "/api/v1/nights/preview"
	if *budget > 0 {
		q := url.Values{}
		q.Set("budget", fmt.Sprintf("%d", *budget))
		path += "?" + q.Encode()
	}

	var plan nightPlan
	if err := apiGet(path, &plan); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
		os.Exit(1)
	}
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(plan)
		return
	}

	fmt.Printf("Window: %s → %s\n", plan.WindowStart, plan.WindowEnd)
	fmt.Printf("Reserved: ~%d tokens", plan.ReservedTokens)
	if plan.BudgetTokens > 0 {
		fmt.Printf(" of %d budget", plan.BudgetTokens)
	}
	fmt.Println()
	fmt.Println()

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "#\tMEMBER\tDUTY\tPRIORITY\tOUTPUT\tCOST\tRISK\tEST.TOKENS")
	for i, s := range plan.Slots {
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%d\n",
			i+1, s.Member, s.Duty, s.Priority, s.Output, s.CostTier, s.Risk, s.EstimatedTokens)
	}
	if err := tw.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
	}
	if len(plan.Skipped) > 0 {
		fmt.Println("\nSkipped:")
		for _, sk := range plan.Skipped {
			fmt.Printf("  %s / %s — %s\n", sk.Member, sk.Duty, sk.Reason)
		}
	}
}
