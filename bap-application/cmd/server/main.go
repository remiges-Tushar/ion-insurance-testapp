package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/indonesiaopennetwork/ion-insurance-testapp/bap-application/internal/db"
	transport "github.com/indonesiaopennetwork/ion-insurance-testapp/bap-application/internal/http"
	"github.com/indonesiaopennetwork/ion-insurance-testapp/bap-application/internal/service"
)

// BAP server entrypoint.
func main() {
	ctx := context.Background()

	// Database
	pool, err := db.NewPool(ctx, db.Config{
		Host:     getenv("BAP_DB_HOST", "localhost"),
		Port:     getenv("BAP_DB_PORT", "5432"),
		User:     getenv("BAP_DB_USER", "insurance"),
		Password: getenv("BAP_DB_PASSWORD", "insurance"),
		DBName:   getenv("BAP_DB_NAME", "insurance_bap"),
		SSLMode:  getenv("BAP_DB_SSLMODE", "disable"),
	})
	if err != nil {
		log.Fatalf("connect to DB: %v", err)
	}
	defer pool.Close()

	// Redis callback manager
	redisAddr := getenv("BAP_REDIS_ADDR", "redis:6379")
	cb := service.NewCallbackManager(redisAddr)

	// onix-bap caller URL
	onixCallerURL := getenv("BAP_ONIX_BAP_CALLER_URL", "http://onix-bap:8081/bap/caller")

	// ION service URL (sole DOKU merchant for VA/QRIS creation and settlement)
	ionServiceURL := getenv("ION_SERVICE_URL", "http://ion:8090")

	// Business logic
	svc := service.NewClientService(pool, cb, onixCallerURL, ionServiceURL)

	// HTTP handlers
	handlers := transport.NewHandlers(svc)

	// Gin engine with CORS
	r := gin.Default()
	r.Use(transport.CORSMiddleware())

	transport.RegisterRoutes(r, handlers)

	port := getenv("BAP_SERVER_PORT", "8083")
	log.Printf("BAP server starting on :%s", port)
	if err := r.Run(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
