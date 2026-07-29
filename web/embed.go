package web

import (
	"embed"
	"io/fs"
)

//go:embed index.html style.css app.js icon.ico
var Files embed.FS

func GetFileSystem() fs.FS {
	return Files
}
