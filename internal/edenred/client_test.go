package edenred_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/niklas/edenred-cli/internal/edenred"
)

func TestFetchBalancesSuccess(t *testing.T) {
	const (
		expectedUser       = "test-user"
		expectedPass       = "test-password"
		sessionToken       = "session-token"
		refreshToken       = "refresh-token"
		lunchBalanceCents  = 6850
		virikeBalanceCents = 12345
		lunchBalance       = float64(lunchBalanceCents) / 100
		virikeBal          = float64(virikeBalanceCents) / 100
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/signin":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method %s for /signin", r.Method)
			}
			var reqBody map[string]string
			if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
				t.Fatalf("decode signin request: %v", err)
			}
			if got := reqBody["username"]; got != expectedUser {
				t.Fatalf("unexpected username %q", got)
			}
			if got := reqBody["password"]; got != expectedPass {
				t.Fatalf("unexpected password %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sessionToken": sessionToken,
				"refreshToken": refreshToken,
			})
		case "/users/me/user-benefits":
			tokenCookie, err := r.Cookie("X-Access-Token")
			if err != nil {
				t.Fatalf("expected X-Access-Token cookie: %v", err)
			}
			if tokenCookie.Value != sessionToken {
				t.Fatalf("unexpected X-Access-Token value %q", tokenCookie.Value)
			}
			refreshCookie, err := r.Cookie("X-Access-Refresh-Token")
			if err != nil {
				t.Fatalf("expected X-Access-Refresh-Token cookie: %v", err)
			}
			if refreshCookie.Value != refreshToken {
				t.Fatalf("unexpected X-Access-Refresh-Token value %q", refreshCookie.Value)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"benefits": []map[string]any{
					{
						"walletType": "main",
						"balance":    lunchBalanceCents,
					},
					{
						"walletType": "wellness",
						"balance":    virikeBalanceCents,
					},
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	client := edenred.NewClient(server.Client(), server.URL)
	ctx := context.Background()
	balances, err := client.FetchBalances(ctx, expectedUser, expectedPass)
	if err != nil {
		t.Fatalf("fetch balances: %v", err)
	}

	if diff := balances.Lunch - lunchBalance; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("unexpected lunch balance %.2f", balances.Lunch)
	}
	if diff := balances.Virike - virikeBal; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("unexpected virike balance %.2f", balances.Virike)
	}
}

func TestFetchBalancesSigninError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/signin" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("invalid credentials"))
	}))
	t.Cleanup(server.Close)

	client := edenred.NewClient(server.Client(), server.URL)
	ctx := context.Background()
	_, err := client.FetchBalances(ctx, "user", "badpass")
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "signin failed"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got %v", want, err)
	}
}

func TestFetchBalancesHandlesDecimalBalance(t *testing.T) {
	const (
		expectedUser = "decimal-user"
		expectedPass = "decimal-password"
		sessionToken = "token"
		refreshToken = "refresh"
		lunchBalance = 12.34
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/signin":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method %s for /signin", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sessionToken": sessionToken,
				"refreshToken": refreshToken,
			})
		case "/users/me/user-benefits":
			tokenCookie, err := r.Cookie("X-Access-Token")
			if err != nil {
				t.Fatalf("expected X-Access-Token cookie: %v", err)
			}
			if tokenCookie.Value != sessionToken {
				t.Fatalf("unexpected X-Access-Token value %q", tokenCookie.Value)
			}
			refreshCookie, err := r.Cookie("X-Access-Refresh-Token")
			if err != nil {
				t.Fatalf("expected X-Access-Refresh-Token cookie: %v", err)
			}
			if refreshCookie.Value != refreshToken {
				t.Fatalf("unexpected X-Access-Refresh-Token value %q", refreshCookie.Value)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"benefits": []map[string]any{
					{
						"walletType": "main",
						"balance":    lunchBalance,
					},
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	client := edenred.NewClient(server.Client(), server.URL)
	ctx := context.Background()
	balances, err := client.FetchBalances(ctx, expectedUser, expectedPass)
	if err != nil {
		t.Fatalf("fetch balances: %v", err)
	}

	if diff := balances.Lunch - lunchBalance; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("unexpected lunch balance %.2f", balances.Lunch)
	}
}
