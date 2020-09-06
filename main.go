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
	orderDelivery
)

func main() {
	app := newApp()
	systray.Run(app.onReady, app.onExit)
}

type Delivery interface {
	RefreshOrderStatus() (OrderStatus, error)
}

func (app *App) onReady() {
	app.noOrder()
	app.refresh()

	mRefresh := systray.AddMenuItem("Refresh order status", "Refresh order status")
	systray.AddSeparator()

	//test purposes only
	mRed := systray.AddMenuItem("Red", "")
	mGreen := systray.AddMenuItem("Green", "")
	mYellow := systray.AddMenuItem("Yellow", "")
	mBlack := systray.AddMenuItem("Black", "")
	mWhite := systray.AddMenuItem("White", "")
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

			case <-mRed.ClickedCh:
				app.orderCreated()
			case <-mGreen.ClickedCh:
				app.orderDelivery()
			case <-mYellow.ClickedCh:
				app.orderCooking()
			case <-mBlack.ClickedCh:
				app.setBlackIcon()
			case <-mWhite.ClickedCh:
				app.setWhiteIcon()

			case <-mQuit.ClickedCh:
				app.quit()
			}

		}
	}()
}

func (app *App) onExit() {
}
