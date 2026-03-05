// Package crescendo embeds static assets and HTML templates into the binary.
package crescendo

import "embed"

//go:embed templates/*.html static/*
var Content embed.FS
