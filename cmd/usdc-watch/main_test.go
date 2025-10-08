package main

import (
	"context"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSendAlertSuccess(t *testing.T) {
	var gotMessage string
	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		gotMessage = req.URL.Query().Get("message")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := sendAlert(ctx, client, "https://example.com/notify", "hello world"); err != nil {
		t.Fatalf("sendAlert returned error: %v", err)
	}
	if gotMessage != "hello world" {
		t.Fatalf("expected message 'hello world', got %q", gotMessage)
	}
}

func TestSendAlertHTTPError(t *testing.T) {
	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusTeapot,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := sendAlert(ctx, client, "https://example.com/notify", "msg"); err == nil {
		t.Fatalf("expected error for non-200 response")
	}
}

func TestBuildAlertMessage(t *testing.T) {
	balance := big.NewInt(1_500_000)
	threshold := big.NewInt(1_000_000)
	msg := buildAlertMessage(balance, threshold)
	expected := "USDC balance 1.500000 >= threshold 1.000000"
	if msg != expected {
		t.Fatalf("buildAlertMessage mismatch: got %q, expected %q", msg, expected)
	}
}
