package openapidocs

import (
	"cmp"
	"embed"
	"html/template"
	"io"
	"io/fs"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
)

//go:embed templates/*
var themes embed.FS

type Config struct {
	// Theme selects a theme
	Theme string
	// Partial true will render only the body content, no <head> etc.
	Partial bool
	// Minify the HTML and CSS output.
	Minify bool
	// EmbedCSS embeds the provided CSS in a <style> tag.
	EmbedCSS string
	// LinkCSS adds a link to an external stylesheet.
	LinkCSS string
}

func Render(output io.Writer, doc *Document, cfg *Config) error {
	cfg = cmp.Or(cfg, new(Config))
	var m *minify.M
	if cfg.Minify {
		m = minify.New()
		m.AddFunc("text/css", css.Minify)
		m.AddFunc("text/html", html.Minify)
	}

	doc.Standalone = !cfg.Partial
	if cfg.EmbedCSS == "" {
		cfg.EmbedCSS = "templates/" + cfg.Theme + ".css"
	}

	cssBytes, err := fs.ReadFile(themes, cfg.EmbedCSS)
	if err != nil {
		return err
	}
	if cfg.Minify {
		cssBytes, err = m.Bytes("text/css", cssBytes)
		if err != nil {
			return err
		}
	}
	doc.StyleContent = template.HTML("<style>" + string(cssBytes) + "</style>")
	if cfg.LinkCSS != "" {
		doc.StyleContent = template.HTML(`<link rel="stylesheet" href="` + cfg.LinkCSS + `">`)
	}
	tmpl, err := template.ParseFS(themes, "templates/"+cfg.Theme+".tmpl")
	if err != nil {
		return err
	}
	if cfg.Minify {
		mw := m.Writer("text/html", output)
		defer mw.Close()
		output = mw
	}
	return tmpl.Execute(output, doc)
}
