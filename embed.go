package main

import "embed"

// The all: prefix is required because Vite emits helper chunks whose names
// begin with an underscore, which directory-pattern embeds skip by default.
//
//go:embed all:frontend/dist
var frontendFS embed.FS
