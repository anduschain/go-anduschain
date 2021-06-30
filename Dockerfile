# Build Geth in a stock Go builder container
FROM golang:1.16-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

ADD . /go-anduschain
RUN cd /go-anduschain && make godaon

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-anduschain/build/bin/godaon /usr/local/bin/

EXPOSE 8545 8546 8555 8565 8575 8556 8566 8576 50505 50515 50525 20535 50505/udp 50515/udp 50525/udp 50535/udp
ENTRYPOINT ["godaon"]
