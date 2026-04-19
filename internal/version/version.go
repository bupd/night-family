// Package version exposes build-time metadata for the nf and nfd binaries.
package version

import (
	"fmt"
	"runtime/debug"
)

// These are set via -ldflags at release build time. They default to
// "dev"/"unknown" for local `go build` invocations and are backfilled from
// debug.ReadBuildInfo where possible.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// Info is the structured form returned by `nf version` and `GET /version`.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	GoVer   string `json:"go"`
}

// Current returns the running binary's version info, backfilling any fields
// left as their default sentinel values from debug.BuildInfo.
func Current() Info {
	v, c, d := Version, Commit, Date
	if bi, ok := debug.ReadBuildInfo(); ok {
		if v == "dev" && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			v = bi.Main.Version
		}
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				if c == "unknown" && s.Value != "" {
					c = s.Value
				}
			case "vcs.time":
				if d == "unknown" && s.Value != "" {
					d = s.Value
				}
			}
		}
	}
	return Info{Version: v, Commit: c, Date: d, GoVer: goVersion(bi())}
}

func bi() *debug.BuildInfo {
	b, _ := debug.ReadBuildInfo()
	return b
}

func goVersion(b *debug.BuildInfo) string {
	if b == nil {
		return "unknown"
	}
	return b.GoVersion
}

// String returns a single-line human rendering: "vX.Y.Z (abc123, 2026-...)".
func (i Info) String() string {
	return fmt.Sprintf("%s (%s, %s, %s)", i.Version, i.Commit, i.Date, i.GoVer)
}
