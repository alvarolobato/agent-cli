FROM golang:1.24-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/agent-cli ./cmd/agent-cli

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /out/agent-cli /agent-cli
USER nonroot:nonroot
ENTRYPOINT ["/agent-cli"]
