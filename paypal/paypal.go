package paypal

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
)

const (
	sandboxURL = "https://api-m.sandbox.paypal.com" // For sandbox testing
	liveURL    = "https://api-m.paypal.com"         // For production
)

type Client struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
	httpClient   *http.Client
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type CreateOrderResponse struct {
	ID string `json:"id"`
}

type CaptureOrderResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Amount struct {
		Total    string `json:"total"`
		Currency string `json:"currency"`
	} `json:"amount"`
}

type CreateOrderRequest struct {
	Amount string `json:"amount"`
}

// NewClient creates a new PayPal client with the specified credentials and mode.
func NewClient(clientID, clientSecret, mode string) *Client {
	baseURL := sandboxURL
	if mode == "live" {
		baseURL = liveURL
	}

	return &Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		BaseURL:      baseURL,
		httpClient:   &http.Client{},
	}
}

// getAccessToken retrieves an access token from PayPal.
func (c *Client) getAccessToken() (string, error) {
	req, err := http.NewRequest("POST", c.BaseURL+"/v1/oauth2/token", nil)
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(c.ClientID, c.ClientSecret)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.URL.RawQuery = "grant_type=client_credentials"

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}

// CreateOrder creates a PayPal order with the specified amount and returns the order ID.
func (c *Client) CreateOrder(amount, currency, returnURL, cancelURL string) (string, error) {
	accessToken, err := c.getAccessToken()
	if err != nil {
		return "", err
	}

	orderPayload := map[string]interface{}{
		"intent": "CAPTURE",
		"purchase_units": []map[string]interface{}{
			{
				"amount": map[string]string{
					"currency_code": currency,
					"value":         amount,
				},
			},
		},
		"application_context": map[string]string{
			"return_url": returnURL,
			"cancel_url": cancelURL,
		},
	}

	payload, _ := json.Marshal(orderPayload)

	req, err := http.NewRequest("POST", c.BaseURL+"/v2/checkout/orders", bytes.NewBuffer(payload))
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var orderResp CreateOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderResp); err != nil {
		return "", err
	}

	return orderResp.ID, nil
}

// CaptureOrder captures the payment for the specified order ID and returns the capture response.
func (c *Client) CaptureOrder(orderID string) (CaptureOrderResponse, error) {
	accessToken, err := c.getAccessToken()
	if err != nil {
		return CaptureOrderResponse{}, err
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/v2/checkout/orders/"+orderID+"/capture", nil)
	if err != nil {
		return CaptureOrderResponse{}, err
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return CaptureOrderResponse{}, err
	}
	defer resp.Body.Close()

	var captureResp CaptureOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&captureResp); err != nil {
		return CaptureOrderResponse{}, err
	}

	return captureResp, nil
}

// CreateOrderHandler handles the HTTP request to create a PayPal order.
func (c *Client) CreateOrderHandler(ctx echo.Context) error {
	var req CreateOrderRequest
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	orderID, err := c.CreateOrder(req.Amount, "USD", "http://your-site.com/return", "http://your-site.com/cancel")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusOK, map[string]string{"id": orderID})
}

// CaptureOrderHandler handles the HTTP request to capture a PayPal order.
func (c *Client) CaptureOrderHandler(ctx echo.Context) error {
	orderID := ctx.QueryParam("orderID")
	if orderID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing orderID")
	}

	orderData, err := c.CaptureOrder(orderID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusOK, orderData)
}

// SetupRoutes sets up the HTTP routes for the PayPal handlers.
func (c *Client) SetupRoutes(e *echo.Echo) {
	e.POST("/create-order", c.CreateOrderHandler)
	e.POST("/capture-order", c.CaptureOrderHandler)
}
