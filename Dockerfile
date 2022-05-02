FROM node:16.14.0 as frontendBuilder
ARG OSS=https://i0.vjoi.cn
ARG SITE=https://ask.vjoi.cn
WORKDIR /work
COPY frontend .
RUN npm install && npm run build


FROM golang:1.17.2 as backendBuilder
WORKDIR /work
COPY . .
RUN go build -o jask cmd/cmd.go

FROM ubuntu:latest
RUN apt-get update && apt-get install -y ca-certificates
WORKDIR /work/
COPY --from=frontendBuilder /work/public ./frontend/public
COPY --from=backendBuilder /work/jask ./
ENV GIN_MODE=release
CMD ["./jask"]
