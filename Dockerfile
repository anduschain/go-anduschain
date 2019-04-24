# Build Geth in a stock Go builder container
FROM golang:1.10-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . /go-anduschain
RUN cd /go-anduschain && make geth

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-anduschain/build/bin/geth /usr/local/bin/

EXPOSE 8545 8546 50505 50505/udp
ENTRYPOINT ["geth"]
