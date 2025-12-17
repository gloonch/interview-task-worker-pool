
FROM golang:1.25.5-alpine AS builder
WORKDIR /src

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=builder /out/app /app/app

EXPOSE 8080
ENTRYPOINT ["/app/app"]
