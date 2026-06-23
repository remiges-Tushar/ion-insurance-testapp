package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("CS_PORT")
	if port == "" {
		port = "9090"
	}
	// BPP webhook base URL — the CS calls the BPP webhook directly because the
	// CS mock is not a registered Beckn network participant and cannot produce a
	// valid signature for the onix receiver's validateSign step.
	bppWebhookURL := os.Getenv("BPP_WEBHOOK_URL")
	if bppWebhookURL == "" {
		bppWebhookURL = "http://bpp:8080/webhook"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// CS receives catalog/publish from onix-bpp caller, validates, returns ACK,
	// then calls back BPP with catalog/on_publish asynchronously.
	mux.HandleFunc("/catalog/publish", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("CS: read body error: %v", err)
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}

		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			log.Printf("CS: unmarshal error: %v", err)
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}

		ctx, _ := req["context"].(map[string]any)
		msg, _ := req["message"].(map[string]any)

		// Extract fields for the on_publish callback
		txnID, _ := ctx["transaction_id"].(string)
		bppID, _ := ctx["bpp_id"].(string)
		domain, _ := ctx["domain"].(string)
		version, _ := ctx["version"].(string)
		if version == "" {
			version = "2.0.0"
		}

		// Extract catalog ID from catalogs[0].id ("CAT-{n}" format) and echo it back
		// in catalog/on_publish so the BPP webhook can update the right DB row.
		var catalogID int64
		if catalogs, ok := msg["catalogs"].([]any); ok && len(catalogs) > 0 {
			if cat, ok := catalogs[0].(map[string]any); ok {
				if id, ok := cat["id"].(string); ok {
					fmt.Sscanf(id, "CAT-%d", &catalogID)
				}
			}
		}

		log.Printf("CS: received catalog/publish bpp_id=%s txn=%s catalog_id=%d", bppID, txnID, catalogID)

		// Return ACK immediately
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]any{
				"ack": map[string]any{"status": "ACK"},
			},
		})

		// Async callback: send catalog/on_publish through beckn-onix BPP receiver.
		// Onix receiver routing maps endpoint 'on_publish' → http://bpp:8080/webhook/catalog/on_publish
		go func() {
			callback := map[string]any{
				"context": map[string]any{
					"action":         "catalog/on_publish",
					"version":        version,
					"domain":         domain,
					"bpp_id":         bppID,
					"transaction_id": txnID,
					"message_id":     txnID + "-onpub",
					"timestamp":      time.Now().UTC().Format(time.RFC3339),
					"ttl":            "PT30S",
				},
				"message": map[string]any{
					"catalog_id": catalogID,
					"catalogs": []map[string]any{
						{
							"bpp_id": bppID,
							"status": "ACCEPTED",
						},
					},
				},
			}
			cbBody, _ := json.Marshal(callback)
			// Call BPP webhook directly — the CS is not a registered Beckn participant
			// so it cannot satisfy the onix receiver's signature validation step.
			url := bppWebhookURL + "/catalog/on_publish"
			resp, err := http.Post(url, "application/json", bytes.NewBuffer(cbBody))
			if err != nil {
				log.Printf("CS: on_publish callback to BPP failed: %v", err)
				return
			}
			defer resp.Body.Close()
			log.Printf("CS: on_publish callback sent to BPP, status=%d", resp.StatusCode)
		}()
	})

	log.Printf("Mock Cataloging Service listening on :%s (BPP webhook: %s)", port, bppWebhookURL)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("CS: server error: %v", err)
	}
}
