package main

import (
	"embed"
	"io/fs"
)

//go:embed all:ui/dist/client
var uiDistClient embed.FS

// uiClientFS returns an fs.FS rooted at ui/dist/client, or nil if the
// embedded directory is empty (build ran before ui was built).
func uiClientFS() fs.FS {
	sub, err := fs.Sub(uiDistClient, "ui/dist/client")
	if err != nil {
		return nil
	}
	return sub
}
