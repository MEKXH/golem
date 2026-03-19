package gateway

import (
	"embed"
	"io/fs"
)

//go:embed all:webui
var embeddedWebUI embed.FS

func webUIFS() (fs.FS, error) {
	return fs.Sub(embeddedWebUI, "webui")
}
