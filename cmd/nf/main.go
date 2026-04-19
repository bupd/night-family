// Command nf is the night-family CLI client.
//
// In this skeleton it exposes a single subcommand, `version`, that prints
// build metadata. Future iterations will add `family`, `duty`, `run`,
// `schedule`, and `budget` subcommands that talk to the nfd daemon over its
// local socket.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/bupd/night-family/internal/version"
)

const usage = `nf — night-family CLI

Usage:
  nf <command> [flags]

Commands:
  version    Print build metadata
  family     Inspect the family roster (list|show)
  duty       Inspect the duty catalogue (list|show)
  help       Show this help

Environment:
  NF_DAEMON_URL   Base URL of the nfd daemon (default http://127.0.0.1:7337)

Run 'nf <command> -h' for command-specific help.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "version", "--version", "-v":
		versionCmd(os.Args[2:])
	case "family":
		familyCmd(os.Args[2:])
	case "duty":
		dutyCmd(os.Args[2:])
	case "help", "--help", "-h":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "nf: unknown command %q\n\n", os.Args[1])
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
}

func versionCmd(args []string) {
	fs := flag.NewFlagSet("version", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "emit JSON instead of a human line")
	_ = fs.Parse(args)

	info := version.Current()
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(info)
		return
	}
	fmt.Println(info.String())
}
