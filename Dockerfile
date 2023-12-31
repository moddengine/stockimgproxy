FROM debian:bookworm-slim as builder

RUN set -xe; \
    apt update && \
    apt install -y build-essential git golang

COPY go.mod go.sum /build/

RUN cd /build && \
    go mod download

COPY *.go /build/

RUN cd /build && \
    go build main && \
    ls -la .

FROM debian:bookworm-slim
RUN apt-get update &&  \
    apt-get install -y ca-certificates &&  \
    apt-get upgrade -y && \
    apt-get clean
WORKDIR /proxy
COPY --from=builder /build/main /usr/bin/stockimgproxy
CMD /usr/bin/stockimgproxy
