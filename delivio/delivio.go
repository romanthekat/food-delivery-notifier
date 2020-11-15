package delivio

import (
	"context"
	"fmt"
	"github.com/EvilKhaosKat/food-delivery-notifier/core"
	fdnHttp "github.com/EvilKhaosKat/food-delivery-notifier/http"
	"math"
	"net/http"
)

type Delivio struct {
	client *fdnHttp.Client
}

type ActiveOrder struct {
	Id     int    `json:"id"`
	Uuid   string `json:"uuid"`
	Status int    `json:"status"`

	Restaurant Restaurant `json:"restaurant"`

	DestLong float32 `json:"longitude"`
	DestLat  float32 `json:"latitude"`
}

type Restaurant struct {
	Id   string         `json:"@id"`
	Type string         `json:"@type"`
	Name string         `json:"name"`
	Info RestaurantInfo `json:"info"`
}

type RestaurantInfo struct {
	Long float32 `json:"longitude"`
	Lat  float32 `json:"latitude"`
}

type CourierCoor struct {
	Long float32 `json:"longitude"`
	Lat  float32 `json:"latitude"`
}

type Orders struct {
	Orders []ActiveOrder `json:"hydra:member"`
}

func NewDelivio(username, password string) (core.Delivery, error) {
	client := fdnHttp.NewHttpClient("https://delivio.by", "/be/api/login", "/be/api/token/refresh")
	response, err := client.Login(context.Background(), &fdnHttp.Login{
		Phone:    username,
		Password: password,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("login response: %+v\n", response)

	return &Delivio{client}, nil
}

func (d *Delivio) RefreshOrderStatus() (core.OrderStatus, string, error) {
	activeOrder, err := d.getActiveOrder()
	if err != nil {
		return core.NoOrder, "", err
	}

	if activeOrder == nil {
		return core.NoOrder, "", nil
	}

	courierCoor, err := d.getCourierCoor(activeOrder.Uuid)
	if err != nil {
		return core.NoOrder, "", err
	}

	restInfo := activeOrder.Restaurant.Info
	switch activeOrder.Status {
	case 2:
		return core.OrderCreated, getDistance(restInfo.Lat, restInfo.Long,
			courierCoor.Lat, courierCoor.Long), nil
	case 4:
		return core.OrderCooking, getDistance(restInfo.Lat, restInfo.Long,
			courierCoor.Lat, courierCoor.Long), nil
	case 16:
		return core.OrderWaitingForDelivery, getDistance(restInfo.Lat, restInfo.Long,
			courierCoor.Lat, courierCoor.Long), nil
	case 12:
		return core.OrderDelivery, getDistance(activeOrder.DestLat, activeOrder.DestLong,
			courierCoor.Lat, courierCoor.Long), nil
	default:
		return core.NoOrder, "", fmt.Errorf("unknown status for order %+v", activeOrder)
	}
}

func (d *Delivio) GetActiveOrder(ctx context.Context) (*ActiveOrder, error) {
	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/be/api/user/orders?orderby[created]=DESC&is_history_viewable=true&itemsPerPage=10&status[]=2&status[]=4&status[]=12&status[]=14&status[]=16",
			d.client.BaseUrl), nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	res := Orders{}
	if err := d.client.SendRequest(req, &res); err != nil {
		return nil, err
	}

	switch ordersCount := len(res.Orders); ordersCount {
	case 0:
		return nil, nil
	case 1:
		return &res.Orders[0], nil
	default:
		return nil, fmt.Errorf("multiple active orders found %+v", res)
	}
}

func (d *Delivio) getCourierCoordinates(ctx context.Context, orderUuid string) ([]CourierCoor, error) {
	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/be/api/order/%s/track", d.client.BaseUrl, orderUuid), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	req = req.WithContext(ctx)

	var res []CourierCoor
	if err := d.client.SendRequest(req, &res); err != nil {
		return nil, err
	}

	return res, nil
}

func (d *Delivio) getActiveOrder() (*ActiveOrder, error) {
	order, err := d.GetActiveOrder(context.Background())
	if err != nil {
		return nil, err
	}

	return order, nil
}

func getDistance(lat1 float32, long1 float32, lat2 float32, long2 float32) string {
	var distanceMeters = 3963.0 * math.Acos(math.Sin(float64(lat1))*math.Sin(float64(lat2))+
		math.Cos(float64(lat1))*math.Cos(float64(lat2))*math.Cos(float64(long2-long1))) * 1.609344 * 1000

	if distanceMeters > 1500 {
		return ">30m"
	} else if distanceMeters > 500 {
		return "20m"
	} else {
		return "5m"
	}
}

func (d *Delivio) getCourierCoor(orderUuid string) (*CourierCoor, error) {
	coordinates, err := d.getCourierCoordinates(context.Background(), orderUuid)
	if err != nil {
		return nil, err
	}

	if len(coordinates) == 0 {
		return nil, fmt.Errorf("courier coordinates are empty")
	}

	//TODO calculate average if coordinates differ
	return &coordinates[0], nil
}
