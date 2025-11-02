FROM golang:1.25.3-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/render-api .

FROM rodorg/rod:latest AS final
WORKDIR /app
COPY --from=builder /app/render-api .
COPY config.yaml .
RUN mkdir screenshots
EXPOSE 8080
ENTRYPOINT ["/app/render-api"]