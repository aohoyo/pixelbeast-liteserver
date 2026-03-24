package admin

import "io/fs"

var staticFS fs.FS

func SetStaticFS(fsys fs.FS) { staticFS = fsys }