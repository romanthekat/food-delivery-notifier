package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseUrl      string
	loginUrl     string
	refreshUrl   string
	accessToken  string
	refreshToken string
	httpClient   *http.Client
}

type Login struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

type ErrorResponse struct {
	HttpCode int
	Code     int    `json:"code"`
	Message  string `json:"message"`
}

func NewHttpClient(baseUrl, loginUrl, refreshUrl string) *Client {
	return &Client{
		baseUrl:    baseUrl,
		loginUrl:   loginUrl,
		refreshUrl: refreshUrl,
		httpClient: &http.Client{
			Timeout: time.Minute,
		},
	}
}

func (c *Client) Login(ctx context.Context, login *Login) (*LoginResponse, error) {
	jsonBytes, err := json.Marshal(login)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseUrl+c.loginUrl, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	res := LoginResponse{}
	if err := c.SendRequest(req, &res); err != nil {
		return nil, err
	}

	c.accessToken = res.AccessToken
	c.refreshToken = res.RefreshToken

	return &res, nil
}

func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	form := url.Values{}
	form.Add("refresh_token", refreshToken)

	req, err := http.NewRequest(http.MethodPost, c.baseUrl+c.refreshUrl, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.PostForm = form
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	req = req.WithContext(ctx)

	res := LoginResponse{}
	if err := c.SendRequest(req, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *Client) SendRequest(req *http.Request, body interface{}) error {
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	for {
		if len(c.accessToken) > 0 {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
		}

		res, err := c.httpClient.Do(req)
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

		//TODO get rid of specific error response format, return whole body
		var errRes ErrorResponse
		if err = json.NewDecoder(res.Body).Decode(&errRes); err == nil {
			errRes.HttpCode = res.StatusCode
			return errRes
		}

		return ErrorResponse{
			HttpCode: res.StatusCode,
			Code:     res.StatusCode,
			Message:  "unknown error\n",
		}
	}
}

func (e ErrorResponse) Error() string {
	return fmt.Sprintf("code: %v, message: %v", e.Code, e.Message)
}
