FROM fidedocker/onix-adapter:1.6.1

RUN apk add --no-cache gettext wget

# Baked in at build time, not fetched at container start — raw.githubusercontent.com
# rate-limits (429) hit us in production more than once, since GitHub Actions/Cloud
# Run egress IPs are shared across many tenants. A stale schema until the next image
# rebuild is a far better failure mode than an unpredictable cold-start 503.
RUN mkdir -p /app/schemas && wget -O /app/schemas/ion-full.yaml \
    https://raw.githubusercontent.com/indonesiaopennetwork/ion-specs/refs/heads/feat/beckn-v2-merge/schema/dist/ion-full.yaml && \
    head -1 /app/schemas/ion-full.yaml | grep -q '^openapi:' || \
    (echo "ERROR: downloaded file doesn't look like the real schema (rate-limited?)" >&2; cat /app/schemas/ion-full.yaml >&2; exit 1)

COPY templates/ /app/templates/
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENV ENVSUBST_VARS='${REDIS_ADDR} ${BAP_URL} ${ONIX_BPP_URL} ${BAP_SIGNING_PRIVATE_KEY} ${BAP_SIGNING_PUBLIC_KEY}'

ENTRYPOINT ["/entrypoint.sh"]
CMD ["sh", "-c", "./server --config=/app/config/insurance-bap.yaml"]
