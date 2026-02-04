FROM node:22 AS frontend-builder
WORKDIR /frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend .
RUN npm run build

FROM node:22 AS admin-builder
WORKDIR /admin
COPY admin/package*.json ./
RUN npm install
COPY admin .
RUN npm run build

FROM golang:1.23 AS backend-builder
WORKDIR /work
COPY . .
RUN go build -o jask cmd/cmd.go

FROM ubuntu:latest
RUN apt-get update && apt-get install -y ca-certificates nodejs npm nginx
WORKDIR /work
COPY --from=frontend-builder /frontend/.next ./frontend/.next
COPY --from=frontend-builder /frontend/public ./frontend/public
COPY --from=frontend-builder /frontend/package*.json ./frontend/
COPY --from=frontend-builder /frontend/node_modules ./frontend/node_modules
COPY --from=admin-builder /admin/.next ./admin/.next
COPY --from=admin-builder /admin/public ./admin/public
COPY --from=admin-builder /admin/package*.json ./admin/
COPY --from=admin-builder /admin/node_modules ./admin/node_modules
COPY --from=backend-builder /work/jask ./
COPY nginx.conf /etc/nginx/nginx.conf
ENV GIN_MODE=release
COPY start.sh ./
RUN chmod +x start.sh
EXPOSE 8080
CMD ["./start.sh"]
