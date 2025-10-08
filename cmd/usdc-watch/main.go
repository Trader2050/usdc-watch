package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"usdc-watch/internal/config"
	"usdc-watch/internal/eth"
	"usdc-watch/internal/rpc"
	"usdc-watch/internal/usdc"
)

func main() {
	cfgPath := flag.String("config", "config/rpc_endpoints.toml", "Path to RPC endpoints configuration")
	addressFlag := flag.String("address", "", "Ethereum wallet address to monitor (hex)")
	thresholdFlag := flag.String("threshold", "", "Alert threshold in USDC (supports up to 6 decimals)")
	intervalFlag := flag.Duration("interval", time.Minute, "Polling interval (e.g. 30s, 1m)")
	onceFlag := flag.Bool("once", false, "Run a single balance check and exit")
	exitAfterAlertFlag := flag.Bool("alert-exit", true, "Exit after the first balance >= threshold alert")
	alertURLFlag := flag.String("alert-url", "", "Optional alert webhook base URL (expects GET with message query param)")

	flag.Parse()

	if *addressFlag == "" {
		log.Fatalf("--address is required")
	}
	if *thresholdFlag == "" {
		log.Fatalf("--threshold is required")
	}
	if *intervalFlag <= 0 {
		log.Fatalf("--interval must be positive")
	}

	normalizedAddress, err := eth.NormalizeAddress(*addressFlag)
	if err != nil {
		log.Fatalf("invalid address: %v", err)
	}

	thresholdAmount, err := usdc.ParseAmount(*thresholdFlag)
	if err != nil {
		log.Fatalf("invalid threshold: %v", err)
	}

	endpoints, err := config.LoadEndpoints(*cfgPath)
	if err != nil {
		log.Fatalf("load endpoints: %v", err)
	}

	rpcClient, err := rpc.NewClient(endpoints, nil)
	if err != nil {
		log.Fatalf("build rpc client: %v", err)
	}

	callData, err := usdc.EncodeBalanceOfCall(normalizedAddress)
	if err != nil {
		log.Fatalf("encode call data: %v", err)
	}

	params := []interface{}{
		map[string]string{
			"to":   usdc.ContractAddress,
			"data": callData,
		},
		"latest",
	}

	pollInterval := *intervalFlag

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Printf("Monitoring USDC balance for %s, threshold %s USDC, interval %s", normalizedAddress, usdc.FormatAmount(thresholdAmount), pollInterval.String())

	runLoop(ctx, logger, rpcClient, params, thresholdAmount, *onceFlag, *exitAfterAlertFlag, pollInterval, *alertURLFlag)
}

func runLoop(
	ctx context.Context,
	logger *log.Logger,
	client *rpc.Client,
	params []interface{},
	threshold *big.Int,
	once bool,
	exitAfterAlert bool,
	interval time.Duration,
	alertURL string,
) {
	for iteration := 0; ; iteration++ {
		if iteration > 0 {
			if once {
				return
			}
			select {
			case <-ctx.Done():
				logger.Printf("Stopping watcher: %v", ctx.Err())
				return
			case <-time.After(interval):
			}
		}

		if err := ctx.Err(); err != nil {
			logger.Printf("Stopping watcher: %v", err)
			return
		}

		iterationCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		balance, endpointName, err := fetchBalance(iterationCtx, client, params)
		cancel()
		if err != nil {
			logger.Printf("Failed to fetch balance: %v", err)
		} else {
			logger.Printf("Balance %s USDC (raw %s) via %s", usdc.FormatAmount(balance), balance.String(), endpointName)
			if balance.Cmp(threshold) >= 0 {
				logger.Printf("ALERT: Balance %s USDC >= threshold %s USDC", usdc.FormatAmount(balance), usdc.FormatAmount(threshold))
				if alertURL != "" {
					alertCtx, cancelAlert := context.WithTimeout(ctx, 5*time.Second)
					if err := sendAlert(alertCtx, http.DefaultClient, alertURL, buildAlertMessage(balance, threshold)); err != nil {
						logger.Printf("Alert webhook failed: %v", err)
					} else {
						logger.Printf("Alert webhook notified")
					}
					cancelAlert()
				}
				if exitAfterAlert {
					return
				}
			}
		}
		if once {
			return
		}
	}
}

func fetchBalance(ctx context.Context, client *rpc.Client, params []interface{}) (*big.Int, string, error) {
	raw, endpoint, err := client.Call(ctx, "eth_call", params)
	if err != nil {
		return nil, endpoint.Name, err
	}
	var hexValue string
	if err := json.Unmarshal(raw, &hexValue); err != nil {
		return nil, endpoint.Name, fmt.Errorf("decode result: %w", err)
	}
	balance, err := hexToBigInt(hexValue)
	if err != nil {
		return nil, endpoint.Name, err
	}
	return balance, endpoint.Name, nil
}

func hexToBigInt(value string) (*big.Int, error) {
	if !strings.HasPrefix(value, "0x") {
		return nil, fmt.Errorf("unexpected hex result: %s", value)
	}
	hexPart := value[2:]
	if hexPart == "" {
		return big.NewInt(0), nil
	}
	amount := new(big.Int)
	if _, ok := amount.SetString(hexPart, 16); !ok {
		return nil, fmt.Errorf("invalid hex value: %s", value)
	}
	return amount, nil
}

func sendAlert(ctx context.Context, client *http.Client, baseURL, message string) error {
	if client == nil {
		client = http.DefaultClient
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("parse alert url: %w", err)
	}
	query := parsed.Query()
	query.Set("message", message)
	parsed.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return fmt.Errorf("build alert request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send alert request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("alert request failed with HTTP %d", resp.StatusCode)
	}
	return nil
}

func buildAlertMessage(balance, threshold *big.Int) string {
	return fmt.Sprintf(
		"USDC balance %s >= threshold %s",
		formatAmountFixed(balance),
		formatAmountFixed(threshold),
	)
}

func formatAmountFixed(amount *big.Int) string {
	formatted := usdc.FormatAmount(amount)
	if strings.Contains(formatted, ".") {
		return formatted
	}
	return formatted + ".000000"
}
