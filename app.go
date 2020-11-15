package main

import (
	"fmt"
	"github.com/getlantern/systray"
	"io/ioutil"
	"syscall"
)

type App struct {
	activeDelivery Delivery
}

func newApp() *App {
	// TODO replace with fyne dialog for user/password if needed
	username, usernameFound := syscall.Getenv("FDN_USERNAME")
	password, passwordFound := syscall.Getenv("FDN_PASSWORD")

	if !usernameFound || !passwordFound {
		panic("username or password not found in env: set FDN_USERNAME and FDN_PASSWORD")
	}

	delivery, err := NewDelivio(username, password)
	if err != nil {
		panic(err)
	}

	return &App{activeDelivery: delivery}
}

func (app *App) refresh() {
	orderStatus, _, err := app.activeDelivery.RefreshOrderStatus()
	if err != nil {
		app.showError(err.Error())
	}

	switch orderStatus {
	case noOrder:
		app.noOrder()
	case orderCreated:
		app.orderCreated()
	case orderCooking:
		app.orderCooking()
	case orderWaitingForDelivery:
		app.orderWaitingForDelivery()
	case orderDelivery:
		app.orderDelivery()
	default:
		panic(fmt.Sprintf("unknown order status detected: %v", orderStatus))
	}
}

func (app *App) quit() {
	systray.Quit()
}

func (app *App) showError(err string) {
	const MaxErrLength = 16
	if len(err) > MaxErrLength {
		err = err[0:15]
	}

	systray.SetTitle(err)
}

func (app *App) orderCreated() {
	systray.SetTooltip("order created")
	systray.SetTitle("")
	app.setIcon("icons/bag/red.png")
}

func (app *App) orderCooking() {
	systray.SetTooltip("cooking")
	systray.SetTitle("")
	app.setIcon("icons/bag/yellow.png")
}

func (app *App) orderWaitingForDelivery() {
	systray.SetTooltip("waiting for delivery")
	systray.SetTitle("")
	app.setIcon("icons/bag/yellow-green.png")
}

func (app *App) orderDelivery() {
	systray.SetTooltip("in delivery")
	systray.SetTitle("")
	app.setIcon("icons/bag/green.png")
}

func (app *App) noOrder() {
	systray.SetTooltip("no active order")
	systray.SetTitle("")
	app.setWhiteIcon()
}

func (app *App) setBlackIcon() {
	app.setIcon("icons/bag/black.png")
}

func (app *App) setWhiteIcon() {
	app.setIcon("icons/bag/white.png")
}

func (app *App) setIcon(icon string) {
	systray.SetIcon(getIcon(icon))
}

func getIcon(s string) []byte {
	b, err := ioutil.ReadFile(s)
	if err != nil {
		fmt.Print(err)
	}
	return b
}
