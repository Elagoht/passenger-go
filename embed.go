//go:build !collage_dev

package main

import "embed"

// files go into the binary, so it runs from any directory. "collage dev"
// builds with embed_dev.go instead and reads them from disk.
//
//go:embed all:templates all:static
var files embed.FS
