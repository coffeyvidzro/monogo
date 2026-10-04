package assets

import (
	"embed"
	"io/fs"
)

//go:embed favicon.ico static
var files embed.FS

func Public() fs.FS {
	return files
}
