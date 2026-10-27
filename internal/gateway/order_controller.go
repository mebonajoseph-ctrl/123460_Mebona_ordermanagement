package gateway

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"order-management/internal/order"
	"order-management/middleware"
)

type OrderController struct {
	svc order.Service
}

func NewOrderController(svc order.Service) *OrderController {
	return &OrderController{svc: svc}
}

// 1. POST /api/v1/orders
func (c *OrderController) CreateOrder(ctx *gin.Context) {
	var req order.CreateOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":     "Invalid input: customerId, productId, and quantity must be positive integers",
			"requestId": middleware.GetRequestID(ctx),
		})
		return
	}

	res, err := c.svc.CreateOrder(ctx.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, order.ErrInvalidInput) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "requestId": middleware.GetRequestID(ctx)})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order", "requestId": middleware.GetRequestID(ctx)})
		return
	}

	ctx.JSON(http.StatusCreated, res)
}

// 2. GET /api/v1/orders/:id
func (c *OrderController) GetOrder(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID format", "requestId": middleware.GetRequestID(ctx)})
		return
	}

	res, err := c.svc.GetOrder(ctx.Request.Context(), id)
	if err != nil {
		if errors.Is(err, order.ErrOrderNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Order not found", "requestId": middleware.GetRequestID(ctx)})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order", "requestId": middleware.GetRequestID(ctx)})
		return
	}

	ctx.JSON(http.StatusOK, res)
}

// 3. GET /api/v1/orders
func (c *OrderController) ListOrders(ctx *gin.Context) {
	orders, err := c.svc.ListOrders(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve orders", "requestId": middleware.GetRequestID(ctx)})
		return
	}

	ctx.JSON(http.StatusOK, orders)
}

// 4. PUT /api/v1/orders/:id/status
func (c *OrderController) UpdateOrderStatus(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID format", "requestId": middleware.GetRequestID(ctx)})
		return
	}

	var req order.UpdateStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: status required", "requestId": middleware.GetRequestID(ctx)})
		return
	}

	if err := c.svc.UpdateOrderStatus(ctx.Request.Context(), id, req.Status); err != nil {
		if errors.Is(err, order.ErrInvalidStatus) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "requestId": middleware.GetRequestID(ctx)})
			return
		}
		if errors.Is(err, order.ErrOrderNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Order not found", "requestId": middleware.GetRequestID(ctx)})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status", "requestId": middleware.GetRequestID(ctx)})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Order status updated successfully",
		"status":  req.Status,
	})
}

// Observability: Health checks
func (c *OrderController) HealthLive(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "UP", "check": "liveness"})
}

func (c *OrderController) HealthReady(ctx *gin.Context) {
	if err := c.svc.CheckHealth(ctx.Request.Context()); err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "READY", "database": "connected"})
}
