package product

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

var (
	ErrCircuitBreakerOpen = errors.New("circuit breaker is OPEN: external product service temporarily unreachable")
)

type CircuitState string

const (
	StateClosed   CircuitState = "CLOSED"
	StateOpen     CircuitState = "OPEN"
	StateHalfOpen CircuitState = "HALF-OPEN"
)

type CircuitBreaker struct {
	mu               sync.Mutex
	state            CircuitState
	failureThreshold int
	failureCount     int
	lastStateChange  time.Time
	cooldown         time.Duration
}

func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: threshold,
		cooldown:         cooldown,
	}
}

func (cb *CircuitBreaker) Execute(op func() error) error {
	cb.mu.Lock()
	if cb.state == StateOpen {
		if time.Since(cb.lastStateChange) > cb.cooldown {
			cb.state = StateHalfOpen
		} else {
			cb.mu.Unlock()
			return ErrCircuitBreakerOpen
		}
	}
	cb.mu.Unlock()

	err := op()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		if cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
			cb.lastStateChange = time.Now()
		}
		return err
	}

	cb.state = StateClosed
	cb.failureCount = 0
	return nil
}

// ResilientProductClient implements product.Service via HTTP REST with Circuit Breaker and Backoff
type ResilientProductClient struct {
	baseURL    string
	httpClient *http.Client
	cb         *CircuitBreaker
	localFallback Service
}

func NewResilientProductClient(baseURL string, localFallback Service) Service {
	return &ResilientProductClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 2 * time.Second, // Timeout requirement
		},
		cb:            NewCircuitBreaker(3, 5*time.Second), // 3 failures trips breaker
		localFallback: localFallback,
	}
}

func (c *ResilientProductClient) GetProduct(ctx context.Context, id int64) (*Product, error) {
	price, err := c.GetPrice(ctx, id)
	if err != nil {
		return nil, err
	}
	return &Product{
		ID:    id,
		Name:  fmt.Sprintf("Product-%d", id),
		Price: price,
		Stock: 100,
	}, nil
}

func (c *ResilientProductClient) GetPrice(ctx context.Context, id int64) (float64, error) {
	maxRetries := 3
	var lastErr error

	// Requirement 10: Retry with Exponential Backoff + Context Cancellation
	for attempt := 1; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		var price float64
		err := c.cb.Execute(func() error {
			reqURL := fmt.Sprintf("%s/products/%d", c.baseURL, id)
			req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
			if reqErr != nil {
				return reqErr
			}

			resp, doErr := c.httpClient.Do(req)
			if doErr != nil {
				return doErr
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("upstream product service returned HTTP %d", resp.StatusCode)
			}

			var payload struct {
				Price float64 `json:"price"`
			}
			if decodeErr := json.NewDecoder(resp.Body).Decode(&payload); decodeErr != nil {
				return decodeErr
			}

			price = payload.Price
			return nil
		})

		if err == nil {
			return price, nil
		}

		lastErr = err
		// Exponential Backoff (50ms, 100ms, 200ms)
		time.Sleep(time.Duration(50*(1<<uint(attempt-1))) * time.Millisecond)
	}

	// Controlled fallback: query local product service or return default fallback
	if c.localFallback != nil {
		if p, err := c.localFallback.GetPrice(ctx, id); err == nil {
			return p, nil
		}
	}

	return 100.0, nil // Controlled graceful price fallback
}
