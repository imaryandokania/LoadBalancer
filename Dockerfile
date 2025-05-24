FROM golang:1.20 as builder

WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o sidecar ./cmd/sidecar

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/sidecar .
EXPOSE 50051
CMD ["./sidecar"]
