package edenred

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.myedenred.fi"

// Client wraps access to the Edenred Finland API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new API client. If httpClient is nil, a client with a sane timeout is used.
func NewClient(httpClient *http.Client, baseURL string) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

// Balances contains the lunch and wellness (Virike) balances.
type Balances struct {
	Lunch  float64
	Virike float64
}

// FetchBalances logs in with the provided credentials and returns the wallet balances.
func (c *Client) FetchBalances(ctx context.Context, username, password string) (*Balances, error) {
	if username == "" || password == "" {
		return nil, errors.New("username and password are required")
	}

	session, err := c.signIn(ctx, username, password)
	if err != nil {
		return nil, fmt.Errorf("signin failed: %w", err)
	}

	benefits, err := c.getUserBenefits(ctx, session.SessionToken, session.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("fetching balances failed: %w", err)
	}

	var balances Balances
	for _, benefit := range benefits {
		amount, err := benefit.balanceFloat64()
		if err != nil {
			return nil, fmt.Errorf("parse %s balance: %w", benefit.WalletType, err)
		}

		switch benefit.WalletType {
		case "main":
			balances.Lunch = amount
		case "wellness":
			balances.Virike = amount
		}
	}

	return &balances, nil
}

type signInRequest struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	RecaptchaToken string `json:"reCaptchaToken"`
}

type signInResponse struct {
	SessionToken string `json:"sessionToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
	Error        string `json:"error"`
	ErrorCode    string `json:"errorCode"`
	FieldName    string `json:"fieldName"`
}

func (c *Client) signIn(ctx context.Context, username, password string) (*signInResponse, error) {
	payload := signInRequest{
		Username:       username,
		Password:       password,
		RecaptchaToken: "",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/signin", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, readRespBody(resp.Body))
	}

	var result signInResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.SessionToken == "" {
		if result.Error != "" {
			return nil, errors.New(result.Error)
		}
		return nil, errors.New("empty session token in response")
	}

	return &result, nil
}

type userBenefit struct {
	CardType           string      `json:"cardType"`
	WalletType         string      `json:"walletType"`
	CardStatus         string      `json:"cardStatus"`
	Balance            json.Number `json:"balance"`
	MobileAvailable    bool        `json:"mobileAvailable"`
	MobilePayment      bool        `json:"mobilePaymentEnabled"`
	ExpectsRenewedCard *string     `json:"expectsRenewedCard"`
	AccountActive      bool        `json:"accountActive"`
}

type userBenefitsResponse struct {
	Benefits []userBenefit `json:"benefits"`
}

func (c *Client) getUserBenefits(ctx context.Context, sessionToken, refreshToken string) ([]userBenefit, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/users/me/user-benefits", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if sessionToken != "" {
		req.AddCookie(&http.Cookie{Name: "X-Access-Token", Value: sessionToken})
	}
	if refreshToken != "" {
		req.AddCookie(&http.Cookie{Name: "X-Access-Refresh-Token", Value: refreshToken})
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, readRespBody(resp.Body))
	}

	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()

	var result userBenefitsResponse
	if err := dec.Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Benefits, nil
}

func (b userBenefit) balanceFloat64() (float64, error) {
	if b.Balance == "" {
		return 0, nil
	}

	if i, err := b.Balance.Int64(); err == nil {
		return float64(i) / 100, nil
	}

	f, err := b.Balance.Float64()
	if err != nil {
		return 0, err
	}

	return f, nil
}

func readRespBody(r io.Reader) string {
	if r == nil {
		return ""
	}
	b, err := io.ReadAll(io.LimitReader(r, 512))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
