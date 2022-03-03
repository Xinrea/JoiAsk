FROM golang:1.17.2
WORKDIR /work
COPY . .
RUN go build -o jask cmd/cmd.go

FROM ubuntu:latest
RUN apt-get update && apt-get install -y ca-certificates
WORKDIR /work/
COPY --from=0 /work/jask ./
CMD ["./jask"]