package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/indonesiaopennetwork/ion-insurance-testapp/bpp-application/internal/service"
)

func RegisterRoutes(r *gin.Engine, h *Handlers, authSvc *service.AuthService) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Auth (public)
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
	}

	// Beckn webhooks (public — called by onix-bpp)
	wh := r.Group("/webhook")
	{
		wh.POST("/discover", h.WebhookSearch)
		wh.POST("/select", h.WebhookSelect)
		wh.POST("/init", h.WebhookInit)
		wh.POST("/confirm", h.WebhookConfirm)
		wh.POST("/status", h.WebhookStatus)
		wh.POST("/cancel", h.WebhookCancel)
		wh.POST("/rate", h.WebhookRate)
		wh.POST("/support", h.WebhookSupport)
		wh.POST("/catalog/on_publish", h.WebhookOnPublish)
	}

	// Protected API routes
	api := r.Group("/api/v1")
	api.Use(AuthMiddleware(authSvc))
	{
		api.GET("/dashboard/stats", h.GetStats)

		api.GET("/policies", h.ListPolicies)
		api.GET("/policies/:id", h.GetPolicy)

		api.GET("/inventory/resources", h.ListResources)
		api.GET("/inventory/offers", h.ListOffers)
		api.GET("/inventory/items", h.ListResources) // alias

		api.GET("/providers", h.ListProviders)
		api.POST("/providers", h.CreateProvider)

		api.GET("/catalogs", h.ListCatalogs)
		api.POST("/catalogs", h.CreateCatalog)
		api.POST("/resources", h.CreateResource)
		api.POST("/offers", h.CreateOffer)
		api.POST("/catalog/publish", h.PublishCatalog)

		api.GET("/claims", h.ListClaims)
		api.GET("/messages", h.ListMessages)
		api.GET("/support-tickets", h.ListSupportTickets)
		api.GET("/ratings", h.ListRatings)
	}
}
