COMPOSE = docker compose
PSQL    = $(COMPOSE) exec -T postgres psql -U insurance
BPP_URL = http://localhost:8080

.PHONY: reset-db seed-admin gen-snap-keys

# Generate a new RSA-2048 key pair for DOKU SNAP QRIS.
# Private key → ion-service/.env (gitignored)
# Public key  → ion-service/snap_public.pem (register this in DOKU merchant portal)
gen-snap-keys:
	@echo "→ Generating RSA-2048 key pair for DOKU SNAP..."
	@openssl genrsa -out /tmp/snap_priv.pem 2048 2>/dev/null
	@openssl rsa -in /tmp/snap_priv.pem -pubout -out ion-service/snap_public.pem 2>/dev/null
	@echo "DOKU_SNAP_PRIVATE_KEY=$$(base64 -w 0 /tmp/snap_priv.pem)" > ion-service/.env
	@rm -f /tmp/snap_priv.pem
	@echo "  ✓ Private key → ion-service/.env (gitignored)"
	@echo "  ✓ Public key  → ion-service/snap_public.pem"
	@echo ""
	@echo "Next: upload ion-service/snap_public.pem to DOKU merchant portal → Settings → API Keys → SNAP"
	@cat ion-service/snap_public.pem

seed-admin:
	@echo "→ Seeding demo admin account..."
	@until curl -sf $(BPP_URL)/health > /dev/null 2>&1 || curl -sf $(BPP_URL)/api/v1/auth/login -X POST -H "Content-Type: application/json" -d '{}' > /dev/null 2>&1; do \
		echo "  waiting for BPP to be ready..."; sleep 2; \
	done
	@curl -s -o /dev/null -w "  register: %{http_code}\n" $(BPP_URL)/api/v1/auth/register \
		-X POST -H "Content-Type: application/json" \
		-d '{"company_name":"PT Asuransi Maju","ojk_license":"OJK-001","email":"admin@asuransimaju.id","password":"password123"}'
	@echo "  demo login: admin@asuransimaju.id / password123"

reset-db:
	@echo "→ Stopping BPP and BAP services..."
	$(COMPOSE) stop bpp bap

	@echo "→ Dropping all tables in insurance_bpp..."
	$(PSQL) -d insurance_bpp -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO insurance;"

	@echo "→ Dropping all tables in insurance_bap..."
	$(PSQL) -d insurance_bap -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO insurance;"

	@echo "→ Running BPP migrations..."
	$(COMPOSE) run --rm migrate-bpp

	@echo "→ Running BAP migrations..."
	$(COMPOSE) run --rm migrate-bap

	@echo "→ Restarting BPP and BAP services..."
	$(COMPOSE) up -d bpp bap

	@echo "→ Restarting frontends to refresh nginx upstream cache..."
	$(COMPOSE) restart bpp-frontend bap-frontend

	@$(MAKE) seed-admin

	@echo "Done - databases reset and migrated."
