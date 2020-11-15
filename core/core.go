package core

type OrderStatus int

const (
	NoOrder OrderStatus = iota
	OrderCreated
	OrderCooking
	OrderWaitingForDelivery
	OrderDelivery
)

type Delivery interface {
	RefreshOrderStatus() (OrderStatus, string, error)
}
