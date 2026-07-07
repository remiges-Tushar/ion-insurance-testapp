package transport

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/indonesiaopennetwork/ion-insurance-testapp/bap-application/internal/api"
	"github.com/indonesiaopennetwork/ion-insurance-testapp/bap-application/internal/service"
)

// Handlers aggregates all HTTP handler methods for the BAP application.
type Handlers struct {
	svc *service.ClientService
}

// NewHandlers creates a Handlers bound to the given ClientService.
func NewHandlers(svc *service.ClientService) *Handlers {
	return &Handlers{svc: svc}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func writeProblem(c *gin.Context, status int, title, detail string) {
	c.JSON(status, api.Problem{Title: title, Detail: detail, Status: status})
}

func ackACK() map[string]any {
	return map[string]any{"message": map[string]any{"ack": map[string]any{"status": "ACK"}}}
}

func ackNACK() map[string]any {
	return map[string]any{"message": map[string]any{"ack": map[string]any{"status": "NACK"}}}
}

// ---------------------------------------------------------------------------
// Frontend-facing handlers
// ---------------------------------------------------------------------------

// Discover handles GET /api/v1/discover?q=<query>
// It proxies directly to onix-bap's discover endpoint (no async wait).
func (h *Handlers) Discover(c *gin.Context) {
	q := c.Query("q")
	result, err := h.svc.Discover(c.Request.Context(), q)
	if err != nil {
		writeProblem(c, http.StatusBadGateway, "Discover Failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

// Search handles POST /api/v1/search
// Body: {"query": "..."}
// Blocks until on_search callback arrives (up to 30 s).
func (h *Handlers) Search(c *gin.Context) {
	var req api.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	result, err := h.svc.Search(c.Request.Context(), req.Query)
	if err != nil {
		writeProblem(c, http.StatusGatewayTimeout, "Search Failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

// Select handles POST /api/v1/select
// Body: arbitrary Beckn select message.
// Blocks until on_select callback arrives (up to 30 s).
func (h *Handlers) Select(c *gin.Context) {
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	result, err := h.svc.Select(c.Request.Context(), req)
	if err != nil {
		writeProblem(c, http.StatusGatewayTimeout, "Select Failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

// Init handles POST /api/v1/init
// Blocks until on_init callback arrives (up to 30 s).
func (h *Handlers) Init(c *gin.Context) {
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	result, err := h.svc.Init(c.Request.Context(), req)
	if err != nil {
		writeProblem(c, http.StatusGatewayTimeout, "Init Failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

// Confirm handles POST /api/v1/confirm
// Blocks until on_confirm callback arrives (up to 30 s).
func (h *Handlers) Confirm(c *gin.Context) {
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	result, err := h.svc.Confirm(c.Request.Context(), req)
	if err != nil {
		writeProblem(c, http.StatusGatewayTimeout, "Confirm Failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetStatus handles GET /api/v1/status/:id
// Non-blocking DB lookup — returns current transaction status + latest snapshot.
func (h *Handlers) GetStatus(c *gin.Context) {
	txnId := c.Param("id")
	result, err := h.svc.GetStatus(c.Request.Context(), txnId)
	if err != nil {
		writeProblem(c, http.StatusNotFound, "Not Found", err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

// RequestStatus handles POST /api/v1/request-status
// Sends a Beckn status request and blocks until on_status arrives (up to 30 s).
func (h *Handlers) RequestStatus(c *gin.Context) {
	var req api.StatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	result, err := h.svc.RequestStatus(c.Request.Context(), req.TransactionID)
	if err != nil {
		writeProblem(c, http.StatusGatewayTimeout, "Status Failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

// Cancel handles POST /api/v1/cancel
// Sends a Beckn cancel request and blocks until on_cancel arrives (up to 30 s).
func (h *Handlers) Cancel(c *gin.Context) {
	var req api.CancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	result, err := h.svc.Cancel(c.Request.Context(), req.TransactionID)
	if err != nil {
		writeProblem(c, http.StatusGatewayTimeout, "Cancel Failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

// Rate handles POST /api/v1/rate
// Blocks until on_rate callback arrives (up to 30 s).
func (h *Handlers) Rate(c *gin.Context) {
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	result, err := h.svc.Rate(c.Request.Context(), req)
	if err != nil {
		writeProblem(c, http.StatusGatewayTimeout, "Rate Failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

// Support handles POST /api/v1/support
// Blocks until on_support callback arrives (up to 30 s).
func (h *Handlers) Support(c *gin.Context) {
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	result, err := h.svc.Support(c.Request.Context(), req)
	if err != nil {
		writeProblem(c, http.StatusGatewayTimeout, "Support Failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

// ListOrders handles GET /api/v1/orders
// Returns all transactions with SEAM stage info.
func (h *Handlers) ListOrders(c *gin.Context) {
	orders, err := h.svc.ListOrders(c.Request.Context())
	if err != nil {
		writeProblem(c, http.StatusInternalServerError, "Internal Error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

// ListPolicies handles GET /api/v1/policies
// Returns all confirmed policy snapshots from the DB.
func (h *Handlers) ListPolicies(c *gin.Context) {
	policies, err := h.svc.ListPolicies(c.Request.Context())
	if err != nil {
		writeProblem(c, http.StatusInternalServerError, "Internal Error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

// GetPolicy handles GET /api/v1/policies/:id
// Returns the confirmed snapshot for the given transaction_id.
func (h *Handlers) GetPolicy(c *gin.Context) {
	txnId := c.Param("id")
	policy, err := h.svc.GetPolicyByTxn(c.Request.Context(), txnId)
	if err != nil {
		writeProblem(c, http.StatusNotFound, "Not Found", err.Error())
		return
	}
	c.JSON(http.StatusOK, policy)
}

// ---------------------------------------------------------------------------
// Beckn callback webhook handlers (on_*)
//
// These are called by onix-bap when a BPP responds.  Each handler must:
//   1. Parse the body.
//   2. Call svc.HandleCallback (which saves the snapshot and publishes to Redis,
//      unblocking the waiting frontend handler).
//   3. Return ACK immediately.
//
// NOTE: HandleCallback is called synchronously (not in a goroutine) because the
// Publish to Redis must complete before the HTTP response is sent — otherwise
// there is a race where the frontend times out before the publish arrives.
// ---------------------------------------------------------------------------

func (h *Handlers) OnSearch(c *gin.Context) {
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, ackNACK())
		return
	}
	if err := h.svc.HandleCallback(c.Request.Context(), "on_discover", payload); err != nil {
		// Log but still ACK — the network should not retry on application errors.
		_ = err
	}
	c.JSON(http.StatusOK, ackACK())
}

func (h *Handlers) OnSelect(c *gin.Context) {
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, ackNACK())
		return
	}
	if err := h.svc.HandleCallback(c.Request.Context(), "on_select", payload); err != nil {
		_ = err
	}
	c.JSON(http.StatusOK, ackACK())
}

func (h *Handlers) OnInit(c *gin.Context) {
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, ackNACK())
		return
	}
	if err := h.svc.HandleCallback(c.Request.Context(), "on_init", payload); err != nil {
		_ = err
	}
	c.JSON(http.StatusOK, ackACK())
}

func (h *Handlers) OnConfirm(c *gin.Context) {
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, ackNACK())
		return
	}
	if err := h.svc.HandleCallback(c.Request.Context(), "on_confirm", payload); err != nil {
		_ = err
	}
	c.JSON(http.StatusOK, ackACK())
}

func (h *Handlers) OnStatus(c *gin.Context) {
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, ackNACK())
		return
	}
	if err := h.svc.HandleCallback(c.Request.Context(), "on_status", payload); err != nil {
		_ = err
	}
	c.JSON(http.StatusOK, ackACK())
}

func (h *Handlers) OnCancel(c *gin.Context) {
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, ackNACK())
		return
	}
	if err := h.svc.HandleCallback(c.Request.Context(), "on_cancel", payload); err != nil {
		_ = err
	}
	c.JSON(http.StatusOK, ackACK())
}

func (h *Handlers) OnRate(c *gin.Context) {
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, ackNACK())
		return
	}
	if err := h.svc.HandleCallback(c.Request.Context(), "on_rate", payload); err != nil {
		_ = err
	}
	c.JSON(http.StatusOK, ackACK())
}

func (h *Handlers) OnSupport(c *gin.Context) {
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, ackNACK())
		return
	}
	if err := h.svc.HandleCallback(c.Request.Context(), "on_support", payload); err != nil {
		_ = err
	}
	c.JSON(http.StatusOK, ackACK())
}

func (h *Handlers) OnReconcile(c *gin.Context) {
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, ackNACK())
		return
	}
	if err := h.svc.HandleCallback(c.Request.Context(), "on_reconcile", payload); err != nil {
		_ = err
	}
	c.JSON(http.StatusOK, ackACK())
}

// PaymentReceived handles POST /api/v1/payment-received
// Called by the frontend on redirect back from DOKU Checkout (?payment=done).
// Triggers ION → BPP payment notification directly, bypassing the external webhook tunnel.
func (h *Handlers) PaymentReceived(c *gin.Context) {
	var req struct {
		TransactionID string `json:"transaction_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.TransactionID == "" {
		writeProblem(c, http.StatusBadRequest, "Bad Request", "transaction_id required")
		return
	}
	if err := h.svc.NotifyPaymentReceived(c.Request.Context(), req.TransactionID); err != nil {
		log.Printf("[BAP] PaymentReceived notify failed: %v", err)
	}
	c.JSON(http.StatusOK, gin.H{"status": "notified"})
}
