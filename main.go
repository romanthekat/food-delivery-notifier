package main

import (
	"github.com/getlantern/systray"
	"time"
)

type OrderStatus int

const (
	noOrder OrderStatus = iota
	orderCreated
	orderCooking
	orderWaitingForDelivery
	orderDelivery
)

func main() {
	app := newApp()
	systray.Run(app.onReady, app.onExit)
}

type Delivery interface {
	RefreshOrderStatus() (OrderStatus, string, error)
}

func (app *App) onReady() {
	app.noOrder()
	app.refresh()

	mRefresh := systray.AddMenuItem("Refresh", "Refresh order status")
	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

	refreshTicker := time.NewTicker(60 * time.Second)

	go func() {
		for {
			select {
			case <-mRefresh.ClickedCh:
				app.refresh()

			case <-refreshTicker.C:
				app.refresh()

			case <-mQuit.ClickedCh:
				app.quit()
			}
		}
	}()
}

func (app *App) onExit() {
}
