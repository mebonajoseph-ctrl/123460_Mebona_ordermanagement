package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"order-management/internal/gateway"
	"order-management/internal/order"
	"order-management/internal/product"
	"order-management/middleware"
)

const testSecret = "jwt-secret-concurrency-test"

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestEnvironment() (*gin.Engine, order.Repository, product.Repository) {
	orderRepo := order.NewMemoryRepository()
	productRepo := product.NewMemoryRepository()

	productSvc := product.NewService(productRepo)
	orderSvc := order.NewService(orderRepo, productSvc)
	orderCtrl := gateway.NewOrderController(orderSvc)

	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())

	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(testSecret))
	{
		api.POST("/orders", orderCtrl.CreateOrder)
		api.GET("/orders/:id", orderCtrl.GetOrder)
		api.GET("/orders", orderCtrl.ListOrders)
		api.PUT("/orders/:id/status", middleware.RequireRole("admin"), orderCtrl.UpdateOrderStatus)
	}

	return r, orderRepo, productRepo
}

func TestOrderCRUDAndSecurity(t *testing.T) {
	r, _, _ := setupTestEnvironment()

	customerToken, _ := middleware.GenerateToken(101, "customer@test.com", "customer", testSecret, time.Hour)
	adminToken, _ := middleware.GenerateToken(1, "admin@test.com", "admin", testSecret, time.Hour)

	// 1. Unauthenticated request -> 401
	body := bytes.NewBufferString(`{"customerId": 101, "productId": 501, "quantity": 2}`)
	reqNoAuth, _ := http.NewRequest(http.MethodPost, "/api/v1/orders", body)
	reqNoAuth.Header.Set("Content-Type", "application/json")
	wNoAuth := httptest.NewRecorder()
	r.ServeHTTP(wNoAuth, reqNoAuth)

	if wNoAuth.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized, got %d", wNoAuth.Code)
	}

	// 2. Create Order with Customer token -> 201 Created
	reqCreate, _ := http.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBufferString(`{"customerId": 101, "productId": 501, "quantity": 2}`))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.Header.Set("Authorization", fmt.Sprintf("Bearer %s", customerToken))
	wCreate := httptest.NewRecorder()
	r.ServeHTTP(wCreate, reqCreate)

	if wCreate.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created, got %d. Body: %s", wCreate.Code, wCreate.Body.String())
	}

	var created order.Order
	if err := json.Unmarshal(wCreate.Body.Bytes(), &created); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if created.ID == 0 || created.TotalPrice <= 0 {
		t.Errorf("Created order verification failed: %+v", created)
	}

	// 3. Customer cannot update status -> 403 Forbidden
	reqForbidden, _ := http.NewRequest(http.MethodPut, "/api/v1/orders/1/status", bytes.NewBufferString(`{"status": "CONFIRMED"}`))
	reqForbidden.Header.Set("Content-Type", "application/json")
	reqForbidden.Header.Set("Authorization", fmt.Sprintf("Bearer %s", customerToken))
	wForbidden := httptest.NewRecorder()
	r.ServeHTTP(wForbidden, reqForbidden)

	if wForbidden.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden, got %d", wForbidden.Code)
	}

	// 4. Admin updates status -> 200 OK
	reqAdmin, _ := http.NewRequest(http.MethodPut, "/api/v1/orders/1/status", bytes.NewBufferString(`{"status": "CONFIRMED"}`))
	reqAdmin.Header.Set("Content-Type", "application/json")
	reqAdmin.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))
	wAdmin := httptest.NewRecorder()
	r.ServeHTTP(wAdmin, reqAdmin)

	if wAdmin.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", wAdmin.Code)
	}
}

// Requirement 11: 100 Concurrent Order Requests without Race Conditions
func TestConcurrent100Orders(t *testing.T) {
	orderRepo := order.NewMemoryRepository()
	productRepo := product.NewMemoryRepository()

	productSvc := product.NewService(productRepo)
	orderSvc := order.NewService(orderRepo, productSvc)

	concurrency := 100
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			req := &order.CreateOrderRequest{
				CustomerID: int64(100 + idx),
				ProductID:  501,
				Quantity:   2,
			}
			o, err := orderSvc.CreateOrder(context.Background(), req)
			if err != nil || o.ID == 0 {
				t.Errorf("Failed concurrent order creation: %v", err)
			}
		}(i)
	}

	wg.Wait()

	list, err := orderRepo.List(context.Background())
	if err != nil {
		t.Fatalf("Failed to retrieve order list: %v", err)
	}
	if len(list) != concurrency {
		t.Errorf("Expected %d orders, got %d", concurrency, len(list))
	}
}
