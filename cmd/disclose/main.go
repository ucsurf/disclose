// Command disclose finds and ranks security reporting channels for a project.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"time"

	"github.com/ucsurf/disclose/internal/disclose"
	"github.com/ucsurf/disclose/internal/report"
)

var version = "dev"

// buildVersion returns the ldflags version, falling back to the module
// version recorded by `go install module@version`.
func buildVersion() string {
	if version != "dev" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return version
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, disclose.New())) }

func run(args []string, stdout, stderr io.Writer, c *disclose.Client) int {
	fs := flag.NewFlagSet("disclose", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "table", "output format: table or json")
	asJSON := fs.Bool("json", false, "shorthand for --format=json")
	verbose := fs.Bool("v", false, "show every check performed")
	timeout := fs.Duration("timeout", 30*time.Second, "overall timeout")
	showVersion := fs.Bool("version", false, "print version")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: disclose [flags] <github/gitlab URL | npm:pkg | pypi:pkg | crates:pkg | domain>\n\nFlags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(stderr, "\nEnv: GITHUB_TOKEN, GITLAB_TOKEN (optional, raise rate limits; PVR status may need a token)\n")
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, "disclose", buildVersion())
		return 0
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}
	if *asJSON {
		*format = "json"
	}
	if *format != "table" && *format != "json" {
		fmt.Fprintf(stderr, "unknown format %q\n", *format)
		return 2
	}
	if c.GitHubToken == "" {
		c.GitHubToken = os.Getenv("GITHUB_TOKEN")
	}
	if c.GitLabToken == "" {
		c.GitLabToken = os.Getenv("GITLAB_TOKEN")
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	res, err := c.Run(ctx, fs.Arg(0))
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	if *format == "json" {
		err = report.JSON(stdout, res)
	} else {
		err = report.Table(stdout, res, *verbose)
	}
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if len(res.Channels) == 0 {
		return 3
	}
	return 0
}
