package transport

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/indonesiaopennetwork/ion-insurance-testapp/bpp-application/internal/api"
	"github.com/indonesiaopennetwork/ion-insurance-testapp/bpp-application/internal/service"
)

type Handlers struct {
	auth    *service.AuthService
	catalog *service.CatalogService
	beckn   *service.BecknService
}

func NewHandlers(auth *service.AuthService, catalog *service.CatalogService, beckn *service.BecknService) *Handlers {
	return &Handlers{auth: auth, catalog: catalog, beckn: beckn}
}

func writeProblem(c *gin.Context, status int, title, detail string) {
	c.JSON(status, api.Problem{Title: title, Detail: detail, Status: status})
}

// Auth

func (h *Handlers) Register(c *gin.Context) {
	var req api.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	acc, err := h.auth.Register(c.Request.Context(), req.CompanyName, req.OJKLicense, req.Email, req.Password)
	if err != nil {
		writeProblem(c, http.StatusConflict, "Registration Failed", err.Error())
		return
	}
	c.JSON(http.StatusCreated, acc)
}

func (h *Handlers) Login(c *gin.Context) {
	var req api.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	token, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		writeProblem(c, http.StatusUnauthorized, "Unauthorized", err.Error())
		return
	}
	c.JSON(http.StatusOK, api.LoginResponse{Token: token})
}

// Dashboard

func (h *Handlers) GetStats(c *gin.Context) {
	stats, err := h.beckn.GetDashboardStats(c.Request.Context())
	if err != nil {
		writeProblem(c, http.StatusInternalServerError, "Internal Error", err.Error())
		return
	}
	c.JSON(http.StatusOK, stats)
}

// Policies

func (h *Handlers) ListPolicies(c *gin.Context) {
	policies, err := h.beckn.ListPolicies(c.Request.Context())
	if err != nil {
		writeProblem(c, http.StatusInternalServerError, "Internal Error", err.Error())
		return
	}
	if policies == nil {
		policies = []service.PolicyRow{}
	}
	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

func (h *Handlers) GetPolicy(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeProblem(c, http.StatusBadRequest, "Bad Request", "invalid id")
		return
	}
	p, err := h.beckn.GetPolicy(c.Request.Context(), id)
	if err != nil {
		writeProblem(c, http.StatusNotFound, "Not Found", "policy not found")
		return
	}
	c.JSON(http.StatusOK, p)
}

// Inventory

func (h *Handlers) ListResources(c *gin.Context) {
	resources, err := h.catalog.ListResources(c.Request.Context())
	if err != nil {
		writeProblem(c, http.StatusInternalServerError, "Internal Error", err.Error())
		return
	}
	if resources == nil {
		resources = []service.ResourceRow{}
	}
	c.JSON(http.StatusOK, gin.H{"resources": resources})
}

func (h *Handlers) ListOffers(c *gin.Context) {
	offers, err := h.catalog.ListOffers(c.Request.Context())
	if err != nil {
		writeProblem(c, http.StatusInternalServerError, "Internal Error", err.Error())
		return
	}
	if offers == nil {
		offers = []service.OfferRow{}
	}
	c.JSON(http.StatusOK, gin.H{"offers": offers})
}

func (h *Handlers) ListCatalogs(c *gin.Context) {
	catalogs, err := h.catalog.ListCatalogs(c.Request.Context())
	if err != nil {
		writeProblem(c, http.StatusInternalServerError, "Internal Error", err.Error())
		return
	}
	if catalogs == nil {
		catalogs = []service.CatalogRow{}
	}
	c.JSON(http.StatusOK, gin.H{"catalogs": catalogs})
}

func (h *Handlers) ListProviders(c *gin.Context) {
	providers, err := h.catalog.ListProviders(c.Request.Context())
	if err != nil {
		writeProblem(c, http.StatusInternalServerError, "Internal Error", err.Error())
		return
	}
	if providers == nil {
		providers = []service.ProviderRow{}
	}
	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

func (h *Handlers) CreateProvider(c *gin.Context) {
	var req api.CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	bppID := "insurance-bpp.iontest"
	if claims, ok := c.Get("claims"); ok {
		if cm, ok := claims.(map[string]any); ok {
			if v, ok := cm["company_name"].(string); ok {
				bppID = v
			}
		}
	}
	if req.Locations == nil {
		req.Locations = []any{}
	}
	id, err := h.catalog.CreateProvider(c.Request.Context(), bppID, req.Name, req.Descriptor, req.Locations, req.ProviderAttributes)
	if err != nil {
		writeProblem(c, http.StatusInternalServerError, "Internal Error", err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handlers) CreateCatalog(c *gin.Context) {
	var req api.CreateCatalogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	claims, _ := c.Get("claims")
	bppID := "insurance-bpp.iontest"
	if cm, ok := claims.(map[string]any); ok {
		if v, ok := cm["company_name"].(string); ok {
			bppID = v
		}
	}
	id, err := h.catalog.CreateCatalog(c.Request.Context(), bppID, req.Name, req.Descriptor, req.Validity, req.Version, req.ProviderID)
	if err != nil {
		writeProblem(c, http.StatusInternalServerError, "Internal Error", err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handlers) CreateResource(c *gin.Context) {
	var req api.CreateResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	bppID := "insurance-bpp.iontest"
	id, err := h.catalog.CreateResource(c.Request.Context(), bppID, req.ProductType, req.VehicleType, req.OJKProductCode, req.ResourceAttributes)
	if err != nil {
		writeProblem(c, http.StatusInternalServerError, "Internal Error", err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handlers) CreateOffer(c *gin.Context) {
	var req api.CreateOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	bppID := "insurance-bpp.iontest"
	id, err := h.catalog.CreateOffer(c.Request.Context(), req.ResourceID, bppID, req.TariffZone, req.PremiumRateMin, req.PremiumRateMax, req.OfferAttributes, req.ValidUntil)
	if err != nil {
		writeProblem(c, http.StatusInternalServerError, "Internal Error", err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handlers) PublishCatalog(c *gin.Context) {
	var req api.PublishCatalogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	bppID := "insurance-bpp.iontest"
	if err := h.catalog.PublishCatalog(c.Request.Context(), req.CatalogID, bppID); err != nil {
		writeProblem(c, http.StatusInternalServerError, "Publish Failed", err.Error())
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "PUBLISH_SENT"})
}

// Claims / Messages / Support / Ratings

func (h *Handlers) ListClaims(c *gin.Context) {
	claims, err := h.beckn.ListClaims(c.Request.Context())
	if err != nil {
		writeProblem(c, http.StatusInternalServerError, "Internal Error", err.Error())
		return
	}
	if claims == nil {
		claims = []service.ClaimRow{}
	}
	c.JSON(http.StatusOK, gin.H{"claims": claims})
}

func (h *Handlers) ListMessages(c *gin.Context) {
	msgs, err := h.beckn.ListMessages(c.Request.Context())
	if err != nil {
		writeProblem(c, http.StatusInternalServerError, "Internal Error", err.Error())
		return
	}
	if msgs == nil {
		msgs = []service.MessageLogRow{}
	}
	c.JSON(http.StatusOK, gin.H{"messages": msgs})
}

func (h *Handlers) ListSupportTickets(c *gin.Context) {
	tickets, err := h.beckn.ListSupportTickets(c.Request.Context())
	if err != nil {
		writeProblem(c, http.StatusInternalServerError, "Internal Error", err.Error())
		return
	}
	if tickets == nil {
		tickets = []service.SupportTicketRow{}
	}
	c.JSON(http.StatusOK, gin.H{"support_tickets": tickets})
}

func (h *Handlers) ListRatings(c *gin.Context) {
	ratings, err := h.beckn.ListRatings(c.Request.Context())
	if err != nil {
		writeProblem(c, http.StatusInternalServerError, "Internal Error", err.Error())
		return
	}
	if ratings == nil {
		ratings = []service.RatingRow{}
	}
	c.JSON(http.StatusOK, gin.H{"ratings": ratings})
}

// Beckn webhook receivers

func (h *Handlers) WebhookSearch(c *gin.Context) {
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ackNACK())
		return
	}
	go h.beckn.HandleSearch(context.Background(), req)
	c.JSON(http.StatusOK, ackACK())
}

func (h *Handlers) WebhookSelect(c *gin.Context) {
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ackNACK())
		return
	}
	go h.beckn.HandleSelect(context.Background(), req)
	c.JSON(http.StatusOK, ackACK())
}

func (h *Handlers) WebhookInit(c *gin.Context) {
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ackNACK())
		return
	}
	go h.beckn.HandleInit(context.Background(), req)
	c.JSON(http.StatusOK, ackACK())
}

func (h *Handlers) WebhookConfirm(c *gin.Context) {
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ackNACK())
		return
	}
	go h.beckn.HandleConfirm(context.Background(), req)
	c.JSON(http.StatusOK, ackACK())
}

func (h *Handlers) WebhookStatus(c *gin.Context) {
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ackNACK())
		return
	}
	go h.beckn.HandleStatus(context.Background(), req)
	c.JSON(http.StatusOK, ackACK())
}

func (h *Handlers) WebhookCancel(c *gin.Context) {
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ackNACK())
		return
	}
	go h.beckn.HandleCancel(context.Background(), req)
	c.JSON(http.StatusOK, ackACK())
}

func (h *Handlers) WebhookRate(c *gin.Context) {
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ackNACK())
		return
	}
	go h.beckn.HandleRate(context.Background(), req)
	c.JSON(http.StatusOK, ackACK())
}

func (h *Handlers) WebhookSupport(c *gin.Context) {
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ackNACK())
		return
	}
	go h.beckn.HandleSupport(context.Background(), req)
	c.JSON(http.StatusOK, ackACK())
}

// Webhook: catalog/on_publish callback from CS (Cataloging Service)
func (h *Handlers) WebhookOnPublish(c *gin.Context) {
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ackNACK())
		return
	}
	msg, _ := req["message"].(map[string]any)

	// catalog_id is echoed back by the CS from the original publish message
	var catalogID int64
	if v, ok := msg["catalog_id"].(float64); ok {
		catalogID = int64(v)
	}

	// status comes from catalogs[0].status per Beckn v2 catalog/on_publish schema
	status := "ACCEPTED"
	if catalogs, ok := msg["catalogs"].([]any); ok && len(catalogs) > 0 {
		if cat, ok := catalogs[0].(map[string]any); ok {
			if s, ok := cat["status"].(string); ok {
				status = s
			}
		}
	}

	h.catalog.HandleOnPublish(c.Request.Context(), catalogID, status, req)
	c.JSON(http.StatusOK, ackACK())
}

// Webhook: DOKU payment notification (VA and QRIS)
func (h *Handlers) HandleDokuWebhook(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}

	clientID := c.GetHeader("Client-Id")
	requestID := c.GetHeader("Request-Id")
	timestamp := c.GetHeader("Request-Timestamp")
	signature := c.GetHeader("Signature")

	if !h.beckn.VerifyDokuWebhook(clientID, requestID, timestamp, "/webhook/doku", bodyBytes, signature) {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid signature"})
		return
	}

	var body map[string]any
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Distinguish VA vs QRIS by service.id in the webhook payload.
	service, _ := body["service"].(map[string]any)
	_ = service // both VA and QRIS share the same invoice_number path

	order, _ := body["order"].(map[string]any)
	invoiceNumber, _ := order["invoice_number"].(string)
	amount, _ := order["amount"].(float64)

	if invoiceNumber == "" {
		c.JSON(http.StatusOK, gin.H{"status": "ignored", "reason": "no invoice_number"})
		return
	}

	if err := h.beckn.ProcessDokuPayment(c.Request.Context(), invoiceNumber, amount); err != nil {
		// Return 200 so DOKU does not retry; error is logged by the service.
		c.JSON(http.StatusOK, gin.H{"status": "received", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// SimulatePayment — sandbox only, marks payment_received=true to simulate DOKU webhook
func (h *Handlers) SimulatePayment(c *gin.Context) {
	txnID := c.Query("txn_id")
	method := c.DefaultQuery("method", "VA") // VA or QRIS
	if txnID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "txn_id required"})
		return
	}
	if err := h.beckn.SimulatePayment(c.Request.Context(), txnID, method); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "simulated", "method": method})
}

func ackACK() map[string]any {
	return map[string]any{"message": map[string]any{"ack": map[string]any{"status": "ACK"}}}
}

func ackNACK() map[string]any {
	return map[string]any{"message": map[string]any{"ack": map[string]any{"status": "NACK"}}}
}

