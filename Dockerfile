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
RUN apt-get update && apt-get install -y ca-certificates nginx
WORKDIR /work
COPY --from=frontend-builder /frontend/out ./frontend
COPY --from=admin-builder /admin/out ./admin
COPY --from=backend-builder /work/jask ./
COPY nginx.conf /etc/nginx/nginx.conf
ENV GIN_MODE=release
COPY start.sh ./
RUN chmod +x start.sh
EXPOSE 80
CMD ["./start.sh"]
