FROM golang:1.27.1-bookworm AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN go build -trimpath -ldflags="-s -w" -o /out/stockimgproxy .

FROM debian:bookworm-slim
RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /proxy
COPY --from=builder /out/stockimgproxy /usr/bin/stockimgproxy
CMD ["/usr/bin/stockimgproxy"]
