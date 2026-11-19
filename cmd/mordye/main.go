// Command mordye completes a sparse theme palette and writes it as JSON on
// stdout.
//
//	mordye < input.json                          mordye's own input format
//	mordye dux --theme FILE [--settings FILE]   a Dux Shell theme
//	mordye --version                            the version it was built from
//	mordye --help                               usage
//
// Mordye reads only the files named on its command line. Finding them (where
// a source keeps its settings, which theme file they select) is the caller's
// job.
//
// It exits 0 on success, 2 when the command itself is wrong (an unknown
// source or flag, or a missing required one), and 1 for anything else, such
// as a file it can't read or a theme it can't complete.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/robwilkerson/mordye"
	"github.com/robwilkerson/mordye/adapter/dux"
)

const usage = `Usage:
  mordye < input.json                          complete mordye's own input format
  mordye dux --theme FILE [--settings FILE]    complete a Dux Shell theme
  mordye --version                             print the version
  mordye --help                                print this help

The complete palette goes to stdout as JSON. Run "mordye dux --help" for its
flags.

Exit status: 0 on success, 2 when the command is wrong, 1 for anything else.
`

func main() {
	if err := dispatch(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "mordye:", err)
		os.Exit(exitCode(err))
	}
}

// usageError is a mistake in the command itself, which the caller can fix,
// rather than in what the command was given.
type usageError struct{ error }

func exitCode(err error) int {
	var u usageError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &u):
		return 2
	default:
		return 1
	}
}

func dispatch(args []string, stdin io.Reader, w io.Writer) error {
	if len(args) == 0 {
		return run(stdin, w)
	}
	switch args[0] {
	case "-h", "--help":
		_, err := io.WriteString(w, usage)
		return err
	case "--version":
		_, err := fmt.Fprintln(w, "mordye", buildVersion())
		return err
	case "dux":
		return runDux(args[1:], w)
	default:
		return usageError{fmt.Errorf("unknown source %q; sources: dux, or no argument to read mordye input on stdin (see --help)", args[0])}
	}
}

// version is stamped by the release build (-ldflags "-X main.version=1.2.3").
var version string

// buildVersion is the stamped version, else the module version Go embeds at
// build time (the tag, for `go install` of a release), else "(devel)".
func buildVersion() string {
	if version != "" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" {
		return bi.Main.Version
	}
	return "(devel)"
}

// run completes Mordye's own input format, read from r.
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
	if err := fs.Parse(args); errors.Is(err, flag.ErrHelp) {
		fmt.Fprint(w, "Usage:\n  mordye dux --theme FILE [--settings FILE]\n\nFlags:\n")
		fs.SetOutput(w)
		fs.PrintDefaults()
		return nil
	} else if err != nil {
		return usageError{fmt.Errorf("dux: %w (see mordye dux --help)", err)}
	}
	if *theme == "" {
		return usageError{fmt.Errorf("dux: --theme is required (see mordye dux --help)")}
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
