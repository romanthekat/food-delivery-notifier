package main

import (
	"github.com/EvilKhaosKat/food-delivery-notifier/app"
	"github.com/getlantern/systray"
)

func main() {
	app := app.NewApp()
	systray.Run(app.OnReady, app.OnExit)
}
