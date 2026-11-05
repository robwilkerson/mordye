// Command mordye completes a sparse theme palette and writes it as JSON on
// stdout.
//
//	mordye < input.json                          mordye's own input format
//	mordye dux --theme FILE [--settings FILE]    a Dux Shell theme
//
// mordye reads only the files named on its command line. Finding them (where
// a source keeps its settings, which theme file they select) is the caller's
// job.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/robwilkerson/mordye"
	"github.com/robwilkerson/mordye/adapter/dux"
)

func main() {
	if err := dispatch(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "mordye:", err)
		os.Exit(1)
	}
}

func dispatch(args []string, stdin io.Reader, w io.Writer) error {
	if len(args) == 0 {
		return run(stdin, w)
	}
	switch args[0] {
	case "dux":
		return runDux(args[1:], w)
	default:
		return fmt.Errorf("unknown source %q; sources: dux, or no argument to read mordye input on stdin", args[0])
	}
}

// run completes mordye's own input format, read from r.
func run(r io.Reader, w io.Writer) error {
	dec := json.NewDecoder(r)
	// A misspelled field would otherwise be dropped without a word, and the
	// palette silently generated without it.
	dec.DisallowUnknownFields()
	var in mordye.Input
	if err := dec.Decode(&in); err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	return write(in, w)
}

func runDux(args []string, w io.Writer) error {
	fs := flag.NewFlagSet("dux", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	theme := fs.String("theme", "", "the theme file the settings select (required)")
	settings := fs.String("settings", "", "Dux's settings.json, for the flavor and accent chosen (default: the theme's own defaults)")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("dux: %w", err)
	}
	if *theme == "" {
		return fmt.Errorf("dux: --theme is required")
	}
	themeBytes, err := os.ReadFile(*theme)
	if err != nil {
		return fmt.Errorf("dux: %w", err)
	}
	var settingsBytes []byte
	if *settings != "" {
		if settingsBytes, err = os.ReadFile(*settings); err != nil {
			return fmt.Errorf("dux: %w", err)
		}
	}
	in, err := dux.Parse(settingsBytes, themeBytes)
	if err != nil {
		return fmt.Errorf("dux: %w", err)
	}
	return write(in, w)
}

func write(in mordye.Input, w io.Writer) error {
	out, err := mordye.Complete(in)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
