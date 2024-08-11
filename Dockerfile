FROM golang:1.22.6-alpine3.20

ARG BUILDOS
ARG TARGETARCH

RUN apk add --no-cache make curl

WORKDIR /app

COPY pkg/ pkg/
COPY cmd/ cmd/

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

RUN CGO_ENABLED=0 GOOS=$BUILDOS GOARCH=$TARGETARCH go build -a -o bin/meilisearch-proxy cmd/meilisearch-proxy/main.go

CMD ["/app/bin/meilisearch-proxy"]