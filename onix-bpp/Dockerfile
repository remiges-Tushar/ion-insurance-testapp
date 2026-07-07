FROM fidedocker/onix-adapter:1.6.1

RUN apk add --no-cache gettext

COPY templates/ /app/templates/
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENV ENVSUBST_VARS='${REDIS_ADDR} ${BPP_URL} ${ONIX_BAP_URL} ${CS_URL} ${BPP_SIGNING_PRIVATE_KEY} ${BPP_SIGNING_PUBLIC_KEY}'

ENTRYPOINT ["/entrypoint.sh"]
CMD ["sh", "-c", "./server --config=/app/config/insurance-bpp.yaml"]
