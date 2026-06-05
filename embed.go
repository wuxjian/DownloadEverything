package main

import "embed"

//go:embed web/templates/*.html web/static/css/*.css web/static/js/*.js
var WebFS embed.FS
