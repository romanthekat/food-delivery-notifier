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
	RefreshOrderStatus() (OrderStatus, Title, error)
}

type Title string
