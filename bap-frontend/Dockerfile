FROM node:20-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/templates/default.conf.template
ENV NGINX_ENVSUBST_FILTER='^(BAP_URL|BPP_URL|ION_URL|NGINX_RESOLVER)$'
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
