// Package web provides the embedded frontend assets.
package web

import "embed"

// Dist embeds the built frontend assets from the dist directory.
//
//go:embed dist/*
var Dist embed.FS
