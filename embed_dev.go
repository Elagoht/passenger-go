//go:build collage_dev

package main

import (
	"io/fs"
	"os"
)

// files is the project directory itself in development.
var files fs.FS = os.DirFS(".")
