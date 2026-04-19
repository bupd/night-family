package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"text/tabwriter"
)

const runUsage = `nf run — inspect and dispatch runs

Usage:
  nf run list [--member X] [--duty Y] [--status Z] [--limit N] [--json]
  nf run show <id>
  nf run start --member X --duty Y        Dispatch a new run against the daemon's provider
`

type runRecord struct {
	ID         string  `json:"id"`
	NightID    *string `json:"night_id,omitempty"`
	Member     string  `json:"member"`
	Duty       string  `json:"duty"`
	Status     string  `json:"status"`
	StartedAt  string  `json:"started_at"`
	FinishedAt *string `json:"finished_at,omitempty"`
	PRURL      *string `json:"pr_url,omitempty"`
	Summary    *string `json:"summary,omitempty"`
}

type runsPage struct {
	Items []runRecord `json:"items"`
}

func runCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, runUsage)
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		runList(args[1:])
	case "show":
		runShow(args[1:])
	case "start":
		runStart(args[1:])
	case "help", "-h", "--help":
		fmt.Print(runUsage)
	default:
		fmt.Fprintf(os.Stderr, "nf run: unknown subcommand %q\n\n", args[0])
		fmt.Fprint(os.Stderr, runUsage)
		os.Exit(2)
	}
}

func runList(args []string) {
	fs := flag.NewFlagSet("run list", flag.ExitOnError)
	member := fs.String("member", "", "filter by member name")
	duty := fs.String("duty", "", "filter by duty type")
	status := fs.String("status", "", "filter by status")
	limit := fs.Int("limit", 0, "max results")
	jsonOut := fs.Bool("json", false, "emit JSON instead of a table")
	_ = fs.Parse(args)

	q := url.Values{}
	if *member != "" {
		q.Set("member", *member)
	}
	if *duty != "" {
		q.Set("duty", *duty)
	}
	if *status != "" {
		q.Set("status", *status)
	}
	if *limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", *limit))
	}
	path := "/api/v1/runs"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}

	var page runsPage
	if err := apiGet(path, &page); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
		os.Exit(1)
	}
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(page.Items)
		return
	}
	if len(page.Items) == 0 {
		fmt.Println("(no runs)")
		return
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tMEMBER\tDUTY\tSTATUS\tSTARTED\tPR")
	for _, r := range page.Items {
		pr := "—"
		if r.PRURL != nil {
			pr = *r.PRURL
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			r.ID, r.Member, r.Duty, r.Status, r.StartedAt, pr)
	}
	_ = tw.Flush()
}

func runStart(args []string) {
	fs := flag.NewFlagSet("run start", flag.ExitOnError)
	member := fs.String("member", "", "family member name (required)")
	duty := fs.String("duty", "", "duty type (required)")
	_ = fs.Parse(args)
	if *member == "" || *duty == "" {
		fmt.Fprintln(os.Stderr, "nf run start: --member and --duty are required")
		os.Exit(2)
	}
	body, err := json.Marshal(map[string]any{"member": *member, "duty": *duty})
	if err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
		os.Exit(1)
	}
	var run map[string]any
	if err := apiPost("/api/v1/runs", body, &run); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(run)
}

func runShow(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "nf run show: <id> required")
		os.Exit(2)
	}
	var body map[string]any
	if err := apiGet("/api/v1/runs/"+args[0], &body); err != nil {
		fmt.Fprintln(os.Stderr, "nf:", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(body)
}
