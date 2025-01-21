package midware

import (
	"log"

	"gopkg.in/kataras/iris.v6"
)

func RouterLog(ctx *iris.Context) {
	// FgBlack Attribute = iota + 30
	// FgRed
	// FgGreen
	// FgYellow
	// FgBlue
	// FgMagenta
	// FgCyan
	// FgWhite

	// 33m: yellow
	log.Printf("\033[33m[REQUEST] %s %s from %s\033[39m", ctx.Method(), ctx.Path(), ctx.RemoteAddr())
	ctx.Next()
}
