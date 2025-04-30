FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
    -ldflags="-w -s" \
    -o ./bot .

FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=builder /app/bot /
USER nonroot:nonroot
ENTRYPOINT ["/bot"]
