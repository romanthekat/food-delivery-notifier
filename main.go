package main

import (
	"github.com/EvilKhaosKat/food-delivery-notifier/app"
	"github.com/getlantern/systray"
)

func main() {
	fdn := app.NewApp()
	systray.Run(fdn.OnReady, fdn.OnExit)
}
