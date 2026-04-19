package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// defaultDaemonURL is the base URL nf uses to reach nfd when no
// --daemon / NF_DAEMON_URL is set.
const defaultDaemonURL = "http://127.0.0.1:7337"

// daemonURL returns the configured nfd base URL — NF_DAEMON_URL takes
// precedence over the built-in default.
func daemonURL() string {
	if u := os.Getenv("NF_DAEMON_URL"); u != "" {
		return u
	}
	return defaultDaemonURL
}

// apiGet fetches and decodes a JSON document from the daemon. Any
// non-2xx response is surfaced as an error with the response body
// inlined (truncated to 512 bytes).
func apiGet(path string, out any) error {
	c := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, daemonURL()+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("reach daemon at %s: %w", daemonURL(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("daemon returned %s: %s", resp.Status, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
