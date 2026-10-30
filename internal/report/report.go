// Package report renders results as a table or JSON.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/ucsurf/disclose/internal/disclose"
)

// JSON writes indented JSON.
func JSON(w io.Writer, r *disclose.Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// Table writes a human-readable table.
func Table(w io.Writer, r *disclose.Result, verbose bool) error {
	fmt.Fprintf(w, "Target: %s\n", r.Target)
	if r.Repo != nil {
		fmt.Fprintf(w, "Repo:   %s\n", r.Repo)
	}
	if r.Homepage != "" {
		fmt.Fprintf(w, "Home:   %s\n", r.Homepage)
	}
	fmt.Fprintln(w)
	if len(r.Channels) == 0 {
		fmt.Fprintln(w, "No security reporting channel found.")
	} else {
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "#\tPRIORITY\tTYPE\tCHANNEL\tSOURCE\tNOTE")
		for i, c := range r.Channels {
			fmt.Fprintf(tw, "%d\t%d\t%s\t%s\t%s\t%s\n", i+1, c.Priority, c.Type, c.Value, c.Source, c.Note)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	}
	if verbose {
		fmt.Fprintln(w, "\nChecks:")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, c := range r.Checks {
			fmt.Fprintf(tw, "  %s\t%s\t%s\n", c.Name, c.Status, strings.TrimSpace(c.Detail))
		}
		return tw.Flush()
	}
	return nil
}
