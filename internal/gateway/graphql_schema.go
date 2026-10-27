package gateway

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
	"order-management/internal/order"
	"order-management/internal/product"
)

func NewGraphQLHandler(orderSvc order.Service, productSvc product.Service) gin.HandlerFunc {
	orderType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Order",
		Fields: graphql.Fields{
			"id":         &graphql.Field{Type: graphql.Int},
			"customerId": &graphql.Field{Type: graphql.Int},
			"productId":  &graphql.Field{Type: graphql.Int},
			"quantity":   &graphql.Field{Type: graphql.Int},
			"totalPrice": &graphql.Field{Type: graphql.Float},
			"status":     &graphql.Field{Type: graphql.String},
		},
	})

	productType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Product",
		Fields: graphql.Fields{
			"id":    &graphql.Field{Type: graphql.Int},
			"name":  &graphql.Field{Type: graphql.String},
			"price": &graphql.Field{Type: graphql.Float},
			"stock": &graphql.Field{Type: graphql.Int},
		},
	})

	rootQuery := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"order": &graphql.Field{
				Type: orderType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id := int64(p.Args["id"].(int))
					return orderSvc.GetOrder(p.Context, id)
				},
			},
			"orders": &graphql.Field{
				Type: graphql.NewList(orderType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return orderSvc.ListOrders(p.Context)
				},
			},
			"product": &graphql.Field{
				Type: productType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id := int64(p.Args["id"].(int))
					return productSvc.GetProduct(p.Context, id)
				},
			},
		},
	})

	rootMutation := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"createOrder": &graphql.Field{
				Type: orderType,
				Args: graphql.FieldConfigArgument{
					"customerId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
					"productId":  &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
					"quantity":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					req := &order.CreateOrderRequest{
						CustomerID: int64(p.Args["customerId"].(int)),
						ProductID:  int64(p.Args["productId"].(int)),
						Quantity:   p.Args["quantity"].(int),
					}
					return orderSvc.CreateOrder(p.Context, req)
				},
			},
		},
	})

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query:    rootQuery,
		Mutation: rootMutation,
	})
	if err != nil {
		log.Fatalf("Failed to initialize GraphQL schema: %v", err)
	}

	h := handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})

	return gin.WrapH(h)
}
