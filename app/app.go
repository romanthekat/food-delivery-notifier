package app

import (
	"fmt"
	"github.com/EvilKhaosKat/food-delivery-notifier/core"
	"github.com/EvilKhaosKat/food-delivery-notifier/delivio"
	"github.com/getlantern/systray"
	"io/ioutil"
	"syscall"
	"time"
)

type App struct {
	activeDelivery core.Delivery
}

func NewApp() *App {
	// TODO replace with fyne or cli dialog for user/password if needed
	username, usernameFound := syscall.Getenv("FDN_USERNAME")
	password, passwordFound := syscall.Getenv("FDN_PASSWORD")

	if !usernameFound || !passwordFound {
		panic("username or password not found in env: set FDN_USERNAME and FDN_PASSWORD")
	}

	delivery, err := delivio.NewDelivio(username, password)
	if err != nil {
		panic(err)
	}

	return &App{activeDelivery: delivery}
}

func (app *App) OnReady() {
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

func (app *App) OnExit() {
}

func (app *App) refresh() {
	orderStatus, title, err := app.activeDelivery.RefreshOrderStatus()
	if err != nil {
		app.showError(err.Error())
	}

	systray.SetTitle(string(title))

	switch orderStatus {
	case core.NoOrder:
		app.noOrder()
	case core.OrderCreated:
		app.orderCreated()
	case core.OrderCooking:
		app.orderCooking()
	case core.OrderWaitingForDelivery:
		app.orderWaitingForDelivery()
	case core.OrderDelivery:
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
	systray.SetTooltip(err)
}

func (app *App) orderCreated() {
	systray.SetTooltip("order created")
	app.setIcon("icons/bag/red.png")
}

func (app *App) orderCooking() {
	systray.SetTooltip("cooking")
	app.setIcon("icons/bag/yellow.png")
}

func (app *App) orderWaitingForDelivery() {
	systray.SetTooltip("waiting for delivery")
	app.setIcon("icons/bag/yellow-green.png")
}

func (app *App) orderDelivery() {
	systray.SetTooltip("in delivery")
	app.setIcon("icons/bag/green.png")
}

func (app *App) noOrder() {
	systray.SetTooltip("no active order")
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
