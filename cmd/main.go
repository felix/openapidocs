package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"userspace.com.au/felix/openapidocs"
)

func main() {
	err := run(os.Args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n%s\n", err)
		os.Exit(1)
	}
}

func run(
	args []string,
	getenv func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	fs := new(flag.FlagSet)
	fs.SetOutput(stderr)
	embedCSS := fs.String("embed-css", "", "Embed CSS from file")
	inPath := fs.String("input", "", "Path to OpenAPI definition")
	isStandalone := fs.Bool("standalone", false, "Create standalone HTML")
	linkCSS := fs.String("link-css", "", "Provide href for CSS file")
	minifyOutput := fs.Bool("minify", false, "Minify output")
	outPath := fs.String("output", "index.html", "Output filename")
	theme := fs.String("theme", "default", "Output theme ")
	useStdin := fs.Bool("stdin", false, "Read OpenAPI definition from stdin")
	useStdout := fs.Bool("stdout", false, "Write to stdout")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	var err error
	var input io.Reader
	var typeHint string
	if *useStdin {
		input = stdin
	} else if *inPath != "" {
		typeHint, _ = strings.CutPrefix(filepath.Ext(*inPath), ".")
		f, err := os.Open(*inPath)
		if err != nil {
			return err
		}
		defer f.Close()
		input = f
	} else {
		return errors.New("no input provided")
	}

	var output io.Writer
	if *useStdout {
		output = stdout
	} else if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			return err
		}
		defer f.Close()
		output = f
	} else {
		return errors.New("no output provided")
	}

	var spec *openapidocs.Spec
	switch strings.ToLower(typeHint) {
	case "yaml", "yml":
		spec, err = openapidocs.ReadSpecFromYAML(input)
	case "json":
		spec, err = openapidocs.ReadSpecFromJSON(input)
	default:
		b, err := io.ReadAll(input)
		if err != nil {
			return err
		}
		if b[0] == '{' {
			spec, err = openapidocs.ParseSpecFromJSON(b)
		}
		if err != nil {
			// FIXME is YAML more common??
			spec, err = openapidocs.ParseSpecFromYAML(b)
		}
	}
	if err != nil {
		return err
	}
	doc, err := openapidocs.DocumentFromSpec(spec)
	if err != nil {
		return err
	}
	cfg := &openapidocs.Config{
		Minify:   *minifyOutput,
		Partial:  !*isStandalone,
		Theme:    *theme,
		EmbedCSS: *embedCSS,
		LinkCSS:  *linkCSS,
	}
	return openapidocs.Render(output, doc, cfg)
}
