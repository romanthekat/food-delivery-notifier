package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Delivio struct {
	accessToken  string
	refreshToken string
	client       *Client
}

func NewDelivio(username, password string) (Delivery, error) {
	client := NewClient("https://delivio.by")
	response, err := client.Login(context.Background(), &Login{
		Phone:    username,
		Password: password,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("login response: %+v\n", response)

	return &Delivio{client: client,
		accessToken: response.AccessToken, refreshToken: response.RefreshToken}, nil
}

func (d Delivio) RefreshOrderStatus() (OrderStatus, error) {
	var activeOrder *ActiveOrder

	for {
		order, err := d.GetActiveOrder(context.Background())
		if err != nil {
			if response, ok := err.(errorResponse); ok {
				if response.Code == http.StatusUnauthorized {
					tokens, err := d.client.RefreshToken(context.Background(), d.refreshToken)
					if err != nil {
						fmt.Printf("error during refreshing token: %s", err)
						return noOrder, err
					}

					d.accessToken = tokens.AccessToken
					d.refreshToken = tokens.RefreshToken
				}
			} else {
				fmt.Printf("error during getting active order: %s", err)
				return noOrder, err
			}
		}

		activeOrder = order
		break
	}

	if activeOrder == nil {
		return noOrder, nil
	}

	switch activeOrder.Status {
	case 2:
		return orderCreated, nil
	case 4:
		return orderCooking, nil
	case 12:
		return orderDelivery, nil
	default:
		return noOrder, fmt.Errorf("unknown status for order %+v", activeOrder)
	}
}

type Orders struct {
	Orders []ActiveOrder `json:"hydra:member"`
}

type ActiveOrder struct {
	Id     string `json:"id"`
	Uuid   string `json:"uuid"`
	Status int    `json:"status"`
}

func (c *Client) Login(ctx context.Context, login *Login) (*LoginResponse, error) {
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
	if err := c.sendRequest(req, "", &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
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
	if err := c.sendRequest(req, "", &res); err != nil {
		return nil, err
	}

	return &res, nil
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
	if err := d.client.sendRequest(req, d.accessToken, &res); err != nil {
		return nil, err
	}

	switch ordersCount := len(res.Orders); ordersCount {
	case 0:
		return nil, nil
	case 1:
		return &res.Orders[0], nil
	default:
		return nil, errors.New(fmt.Sprintf("multiple active orders found %+v", res))
	}
}

type Client struct {
	baseUrl string
	client  *http.Client
}

func NewClient(baseUrl string) *Client {
	return &Client{
		baseUrl: baseUrl,
		client: &http.Client{
			Timeout: time.Minute,
		},
	}
}

type errorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e errorResponse) Error() string {
	return fmt.Sprintf("code: %v, message: %v", e.Code, e.Message)
}

type Login struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

func (c *Client) sendRequest(req *http.Request, accessToken string, v interface{}) error {
	if len(req.Header.Get("Content-Type")) == 0 {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	if len(accessToken) > 0 {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	}

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		var errRes errorResponse
		if err = json.NewDecoder(res.Body).Decode(&errRes); err == nil {
			return errors.New(fmt.Sprintf("error code: %v, message: %s",
				errRes.Code, errRes.Message))
		}

		return errorResponse{
			Code:    res.StatusCode,
			Message: "unknown error",
		}
	}

	if err = json.NewDecoder(res.Body).Decode(&v); err != nil {
		return err
	}

	return nil
}
