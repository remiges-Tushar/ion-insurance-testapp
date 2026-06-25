package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires all routes onto the Gin engine.
func RegisterRoutes(r *gin.Engine, h *Handlers) {
	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Frontend-facing API (no auth — the BAP is a customer-facing service)
	api := r.Group("/api/v1")
	{
		api.GET("/discover", h.Discover)
		api.POST("/discover", h.Search)
		api.POST("/select", h.Select)
		api.POST("/init", h.Init)
		api.POST("/confirm", h.Confirm)

		// Non-blocking DB status lookup
		api.GET("/status/:id", h.GetStatus)
		// Blocking Beckn status request
		api.POST("/request-status", h.RequestStatus)

		api.POST("/cancel", h.Cancel)
		api.POST("/rate", h.Rate)
		api.POST("/support", h.Support)

		api.GET("/policies", h.ListPolicies)
		api.GET("/policies/:id", h.GetPolicy)
	}

	// Beckn callback webhooks (called by onix-bap when BPP responds)
	wh := r.Group("/webhook")
	{
		wh.POST("/on_discover", h.OnSearch)
		wh.POST("/on_select", h.OnSelect)
		wh.POST("/on_init", h.OnInit)
		wh.POST("/on_confirm", h.OnConfirm)
		wh.POST("/on_status", h.OnStatus)
		wh.POST("/on_cancel", h.OnCancel)
		wh.POST("/on_rate", h.OnRate)
		wh.POST("/on_support", h.OnSupport)
		wh.POST("/on_reconcile", h.OnReconcile)
	}
}
