package delivio

import (
	"context"
	"fmt"
	"github.com/EvilKhaosKat/food-delivery-notifier/core"
	fdnHttp "github.com/EvilKhaosKat/food-delivery-notifier/http"
	"math"
	"net/http"
	"strconv"
)

const (
	baseUrl       = "https://delivio.by"
	loginUrl      = "/be/api/login"
	refreshUrl    = "/be/api/token/refresh"
	earthRadiusKm = 6371
)

type Delivio struct {
	client *fdnHttp.Client
}

type ActiveOrder struct {
	Id       int    `json:"id"`
	Uuid     string `json:"uuid"`
	StatusId int    `json:"status"`

	TotalPrice float32 `json:"totalPrice"`

	Restaurant Restaurant `json:"restaurant"`

	DestLong float32 `json:"longitude"`
	DestLat  float32 `json:"latitude"`
}

type Restaurant struct {
	Id   string         `json:"@id"`
	Type string         `json:"@type"`
	Name string         `json:"name"`
	Info RestaurantInfo `json:"Info"`
}

type RestaurantInfo struct {
	Address *Coor `json:"address"`
}

type Coor struct {
	Long float32 `json:"longitude"`
	Lat  float32 `json:"latitude"`
}

type Orders struct {
	Orders []ActiveOrder `json:"hydra:member"`
}

func NewDelivio(username, password string) (core.Delivery, error) {
	client := fdnHttp.NewHttpClient(baseUrl, loginUrl, refreshUrl)
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

func (d *Delivio) RefreshOrderStatus() (core.OrderStatus, core.Title, error) {
	activeOrder, err := d.getActiveOrder(context.Background())
	if err != nil {
		return core.NoOrder, "", err
	}

	if activeOrder == nil {
		writeLogs(&logData{
			restaurantId: "",
			status:       "",
			totalPrice:   0.0,
			restCoor:     "",
			destCoor:     "",
			courierCoor:  "",
		})
		return core.NoOrder, "", nil
	}

	courierCoor, err := d.getCourierCoor(activeOrder.Uuid)
	if err != nil {
		return core.NoOrder, "", err
	}

	restCoor := &Coor{activeOrder.Restaurant.Info.Address.Long, activeOrder.Restaurant.Info.Address.Lat}
	destCoor := &Coor{activeOrder.DestLong, activeOrder.DestLat}

	status, err := getStatusById(activeOrder.StatusId)
	if err != nil {
		return core.NoOrder, "", err
	}

	title, err := getTitleByStatus(status, restCoor, destCoor, courierCoor)
	if err != nil {
		return status, "", err
	}

	writeLogs(&logData{
		restaurantId: activeOrder.Restaurant.Id,
		status:       strconv.Itoa(int(status)),
		totalPrice:   activeOrder.TotalPrice,
		restCoor:     fmt.Sprintf("%+v", restCoor),
		destCoor:     fmt.Sprintf("%+v", destCoor),
		courierCoor:  fmt.Sprintf("%+v", courierCoor),
	})

	return status, title, nil
}

func getTitleByStatus(status core.OrderStatus, restCoor, destCoor, courierCoor *Coor) (core.Title, error) {
	switch status {
	case core.OrderCreated:
		return getTitle(courierCoor, restCoor), nil
	case core.OrderCooking:
		return getTitle(courierCoor, restCoor), nil
	case core.OrderWaitingForDelivery:
		return getTitle(courierCoor, restCoor), nil
	case core.OrderDelivery:
		return getTitle(courierCoor, destCoor), nil
	default:
		return "", fmt.Errorf("unknown status for status %+v", status)
	}
}

func getStatusById(status int) (core.OrderStatus, error) {
	switch status {
	case 2:
		return core.OrderCreated, nil
	case 4:
		return core.OrderCooking, nil
	case 16:
		return core.OrderWaitingForDelivery, nil
	case 12:
		return core.OrderDelivery, nil
	default:
		return core.NoOrder, fmt.Errorf("unknown status for order %+v", status)
	}
}

func getTitle(courier, dest *Coor) core.Title {
	if courier == nil {
		return "no courier"
	}

	title, err := getDistance(courier, dest)
	if err != nil {
		return core.Title(err.Error())
	}

	return title
}

func (d *Delivio) getActiveOrder(ctx context.Context) (*ActiveOrder, error) {
	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/be/api/user/orders?orderby[created]=DESC&is_history_viewable=true&itemsPerPage=10&status[]=2&status[]=4&status[]=12&status[]=14&status[]=16",
			baseUrl), nil)
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

func (d *Delivio) getCourierCoordinates(ctx context.Context, orderUuid string) ([]Coor, error) {
	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/be/api/order/%s/track", baseUrl, orderUuid), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	req = req.WithContext(ctx)

	var res []Coor
	if err := d.client.SendRequest(req, &res); err != nil {
		return nil, err
	}

	return res, nil
}

func getDistance(coor1, coor2 *Coor) (core.Title, error) {
	if coor1 == nil || coor2 == nil {
		return "", fmt.Errorf("both coordinates must be provided")
	}

	fmt.Printf("coor1:%+v, coor2: %+v\n", coor1, coor2)

	//haversine formula
	lat1 := toRadians(coor1.Lat)
	lon1 := toRadians(coor1.Long)
	lat2 := toRadians(coor2.Lat)
	lon2 := toRadians(coor2.Long)

	deltaLat := lat2 - lat1
	deltaLong := lon2 - lon1

	a := math.Sin(float64(deltaLat/2))*math.Sin(float64(deltaLat/2)) + math.Cos(float64(lat1))*math.Cos(float64(lat2))*
		math.Sin(float64(deltaLong/2))*math.Sin(float64(deltaLong/2))

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distanceMeters := c * earthRadiusKm * 1000

	fmt.Printf("distance: %f\n", distanceMeters)

	if distanceMeters > 1500 {
		return ">30m", nil
	} else if distanceMeters > 500 {
		return "20m", nil
	} else {
		return "5m", nil
	}
}

func toRadians(d float32) float32 {
	return d * math.Pi / 180
}

func (d *Delivio) getCourierCoor(orderUuid string) (*Coor, error) {
	coordinates, err := d.getCourierCoordinates(context.Background(), orderUuid)
	if err != nil {
		return nil, err
	}

	if len(coordinates) == 0 {
		return nil, nil
	}

	//TODO calculate average if coordinates differ
	return &coordinates[0], nil
}
