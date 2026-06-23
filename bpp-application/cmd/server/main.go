package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	transport "github.com/indonesiaopennetwork/ion-insurance-testapp/bpp-application/internal/http"
	"github.com/indonesiaopennetwork/ion-insurance-testapp/bpp-application/internal/db"
	"github.com/indonesiaopennetwork/ion-insurance-testapp/bpp-application/internal/service"
)

func main() {
	ctx := context.Background()

	pool, err := db.NewPool(ctx, db.Config{
		Host:     getenv("BPP_DB_HOST", "localhost"),
		Port:     getenv("BPP_DB_PORT", "5432"),
		User:     getenv("BPP_DB_USER", "insurance"),
		Password: getenv("BPP_DB_PASSWORD", "insurance"),
		DBName:   getenv("BPP_DB_NAME", "insurance_bpp"),
		SSLMode:  getenv("BPP_DB_SSLMODE", "disable"),
	})
	if err != nil {
		log.Fatalf("connect to DB: %v", err)
	}
	defer pool.Close()

	authSvc := service.NewAuthService(pool)
	catalogSvc := service.NewCatalogService(pool)
	becknSvc := service.NewBecknService(pool)

	handlers := transport.NewHandlers(authSvc, catalogSvc, becknSvc)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Type", "Accept", "Authorization"},
		MaxAge:          12 * time.Hour,
	}))

	transport.RegisterRoutes(r, handlers, authSvc)

	port := getenv("BPP_SERVER_PORT", "8080")
	log.Printf("BPP server starting on :%s", port)
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
