package delivio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/EvilKhaosKat/food-delivery-notifier/core"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Delivio struct {
	client *HttpClient
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

type Login struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

type CourierCoor struct {
	Long float32 `json:"longitude"`
	Lat  float32 `json:"latitude"`
}

type Orders struct {
	Orders []ActiveOrder `json:"hydra:member"`
}

type HttpClient struct {
	baseUrl      string
	accessToken  string
	refreshToken string
	client       *http.Client
}

type errorResponse struct {
	HttpCode int
	Code     int    `json:"code"`
	Message  string `json:"message"`
}

func NewDelivio(username, password string) (core.Delivery, error) {
	client := NewHttpClient("https://delivio.by")
	response, err := client.Login(context.Background(), &Login{
		Phone:    username,
		Password: password,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("login response: %+v\n", response)

	client.accessToken = response.AccessToken
	client.refreshToken = response.RefreshToken

	return &Delivio{client}, nil
}

func NewHttpClient(baseUrl string) *HttpClient {
	return &HttpClient{
		baseUrl: baseUrl,
		client: &http.Client{
			Timeout: time.Minute,
		},
	}
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
			d.client.baseUrl), nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	res := Orders{}
	if err := d.client.sendRequest(req, &res); err != nil {
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

func (c *HttpClient) Login(ctx context.Context, login *Login) (*LoginResponse, error) {
	jsonBytes, err := json.Marshal(login)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/be/api/login", c.baseUrl), bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	res := LoginResponse{}
	if err := c.sendRequest(req, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *HttpClient) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	form := url.Values{}
	form.Add("refresh_token", refreshToken)

	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/be/api/token/refresh", c.baseUrl), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.PostForm = form
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	req = req.WithContext(ctx)

	res := LoginResponse{}
	if err := c.sendRequest(req, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *HttpClient) GetCourierCoordinates(ctx context.Context, orderUuid string) ([]CourierCoor, error) {
	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/be/api/order/%s/track", c.baseUrl, orderUuid), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	req = req.WithContext(ctx)

	var res []CourierCoor
	if err := c.sendRequest(req, &res); err != nil {
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
	coordinates, err := d.client.GetCourierCoordinates(context.Background(), orderUuid)
	if err != nil {
		return nil, err
	}

	if len(coordinates) == 0 {
		return nil, fmt.Errorf("courier coordinates are empty")
	}

	//TODO calculate average if coordinates differ
	return &coordinates[0], nil
}

func (c *HttpClient) sendRequest(req *http.Request, body interface{}) error {
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	for {
		if len(c.accessToken) > 0 {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
		}

		res, err := c.client.Do(req)
		if err != nil {
			return err
		}

		defer res.Body.Close()

		if res.StatusCode == http.StatusOK {
			err := json.NewDecoder(res.Body).Decode(&body)
			if err != nil {
				return err
			}
			return nil
		}

		if res.StatusCode == http.StatusUnauthorized {
			tokens, err := c.RefreshToken(context.Background(), c.refreshToken)
			if err != nil {
				fmt.Printf("error during refreshing token: %s\n", err)
				return err
			}

			c.accessToken = tokens.AccessToken
			c.refreshToken = tokens.RefreshToken

			continue
		}

		//TODO get rid of specific error response format
		var errRes errorResponse
		if err = json.NewDecoder(res.Body).Decode(&errRes); err == nil {
			errRes.HttpCode = res.StatusCode
			return errRes
		}

		return errorResponse{
			HttpCode: res.StatusCode,
			Code:     res.StatusCode,
			Message:  "unknown error",
		}
	}
}

func (e errorResponse) Error() string {
	return fmt.Sprintf("code: %v, message: %v", e.Code, e.Message)
}
