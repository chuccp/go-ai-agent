package main

import (
	"embed"
	"io/fs"
)

//go:embed view/dist/*
var embeddedAssets embed.FS

// assetFS returns the embedded frontend assets (nil in dev mode if no dist built).
func assetFS() fs.FS {
	sub, err := fs.Sub(embeddedAssets, "view/dist")
	if err != nil {
		return nil
	}
	return sub
}
