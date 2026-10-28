package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"order-management/internal/order"
	"order-management/internal/product"
)

func setupTestRouter(t *testing.T) (*gin.Engine, order.Repository, product.Repository) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Use in-memory repositories to decouple tests from external PostgreSQL/RDS
	orderRepo := order.NewMemoryRepository()
	prodRepo := product.NewMemoryRepository()

	// Seed product inventory with ample stock for concurrent order tests
	ctx := context.Background()
	_ = prodRepo.Save(ctx, &product.Product{
		ID:    501,
		Name:  "Test High-Volume Product",
		Price: 99.99,
		Stock: 500,
	})

	prodSvc := product.NewService(prodRepo)
	orderSvc := order.NewService(orderRepo, prodSvc)
	orderCtrl := order.NewController(orderSvc)

	api := router.Group("/api/v1")
	{
		api.POST("/orders", orderCtrl.CreateOrder)
		api.GET("/orders/:id", orderCtrl.GetOrder)
		api.GET("/orders", orderCtrl.ListOrders)
		api.PUT("/orders/:id/status", orderCtrl.UpdateOrderStatus)
	}

	return router, orderRepo, prodRepo
}

func TestOrderConcurrency(t *testing.T) {
	router, orderRepo, _ := setupTestRouter(t)

	const totalRequests = 100
	var wg sync.WaitGroup
	var successfulRequests int64
	var failedRequests int64

	payload := map[string]interface{}{
		"customerId": 101,
		"productId":  501,
		"quantity":   2,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal JSON payload: %v", err)
	}

	// Dispatch 100 concurrent order requests
	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req, reqErr := http.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBuffer(bodyBytes))
			if reqErr != nil {
				atomic.AddInt64(&failedRequests, 1)
				return
			}
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusCreated {
				atomic.AddInt64(&successfulRequests, 1)
			} else {
				atomic.AddInt64(&failedRequests, 1)
			}
		}()
	}

	wg.Wait()

	// Assertions for concurrent request execution
	if failedRequests > 0 {
		t.Errorf("Expected 0 failures across 100 concurrent requests, got %d failures", failedRequests)
	}

	if successfulRequests != totalRequests {
		t.Fatalf("Expected %d successful creations, got %d", totalRequests, successfulRequests)
	}

	// Verify all 100 records were persisted safely in the repository
	ctx := context.Background()
	orders, listErr := orderRepo.List(ctx)
	if listErr != nil {
		t.Fatalf("Failed to fetch orders from repository after concurrency test: %v", listErr)
	}

	if len(orders) != totalRequests {
		t.Errorf("Repository contains %d orders, expected %d", len(orders), totalRequests)
	}
}

func TestConcurrentOrderReadWrite(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	// Pre-create one order to test concurrent read/write race conditions
	createPayload := `{"customerId": 200, "productId": 501, "quantity": 1}`
	initReq, _ := http.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBufferString(createPayload))
	initReq.Header.Set("Content-Type", "application/json")
	initResp := httptest.NewRecorder()
	router.ServeHTTP(initResp, initReq)

	if initResp.Code != http.StatusCreated {
		t.Fatalf("Failed to create seed order: status %d", initResp.Code)
	}

	var createdOrder struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(initResp.Body.Bytes(), &createdOrder); err != nil {
		t.Fatalf("Failed to parse created order ID: %v", err)
	}

	var wg sync.WaitGroup
	const iterations = 50

	// Half goroutines reading orders, half creating new orders
	for i := 0; i < iterations; i++ {
		wg.Add(2)

		// Concurrent reader
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/orders", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Concurrent GET /orders returned status %d", w.Code)
			}
		}()

		// Concurrent writer
		go func() {
			defer wg.Done()
			body := bytes.NewBufferString(`{"customerId": 300, "productId": 501, "quantity": 1}`)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/orders", body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				t.Errorf("Concurrent POST /orders returned status %d", w.Code)
			}
		}()
	}

	wg.Wait()
}