package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/niklas/edenred-cli/internal/edenred"
)

func main() {
	username := flag.String("username", "", "Edenred username")
	password := flag.String("password", "", "Edenred password")
	format := flag.String("format", "text", "Output format: text or json")
	baseURL := flag.String("base-url", "", "Override API base URL (for testing)")
	timeout := flag.Duration("timeout", 15*time.Second, "Request timeout")
	flag.Parse()

	if err := run(*username, *password, *format, *baseURL, *timeout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(username, password, format, baseURL string, timeout time.Duration) error {
	if username == "" || password == "" {
		return fmt.Errorf("username and password are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := edenred.NewClient(nil, baseURL)
	balances, err := client.FetchBalances(ctx, username, password)
	if err != nil {
		return err
	}

	switch strings.ToLower(format) {
	case "text":
		fmt.Printf("Lunch: %.2f\n", balances.Lunch)
		fmt.Printf("Virike: %.2f\n", balances.Virike)
	case "json":
		payload := map[string]float64{
			"lunch":  balances.Lunch,
			"virike": balances.Virike,
		}
		if err := json.NewEncoder(os.Stdout).Encode(payload); err != nil {
			return fmt.Errorf("encode json: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format %q", format)
	}

	return nil
}
