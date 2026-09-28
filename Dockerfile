FROM node:22 AS frontend-builder
WORKDIR /workspace/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend .
COPY shared /workspace/shared
RUN npm run build

FROM node:22 AS admin-builder
WORKDIR /workspace/admin
COPY admin/package*.json ./
RUN npm ci
COPY admin .
COPY shared /workspace/shared
RUN npm run build

FROM golang:1.25-alpine AS backend-builder
WORKDIR /work
RUN apk add --no-cache gcc musl-dev
ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE"
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /jask ./cmd/cmd.go

FROM caddy:2-alpine
WORKDIR /work
COPY --from=frontend-builder /workspace/frontend/out ./frontend
COPY --from=admin-builder /workspace/admin/out ./admin
COPY --from=backend-builder /jask ./
COPY Caddyfile /etc/caddy/Caddyfile
ENV GIN_MODE=release
COPY start.sh /usr/local/bin/start-joiask
RUN mkdir -p /work/frontend/public/upload-img \
	&& chmod +x /usr/local/bin/start-joiask
EXPOSE 80
CMD ["start-joiask"]
