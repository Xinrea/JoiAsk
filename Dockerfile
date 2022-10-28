FROM node:16.14.0 as frontendBuilder
WORKDIR /work
COPY frontend .
RUN npm install && npm run build

FROM golang:1.18 as backendBuilder
WORKDIR /work
COPY . .
RUN go build -o jask cmd/cmd.go

FROM ubuntu:latest
RUN apt-get update && apt-get install -y ca-certificates
WORKDIR /work/
COPY --from=frontendBuilder /work/dist ./frontend/public
COPY --from=backendBuilder /work/jask ./
ENV GIN_MODE=release
CMD ["./jask"]
