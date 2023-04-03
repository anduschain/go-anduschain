# Build Geth in a stock Go builder container
FROM golang:1.18-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

ADD . /go-anduschain
RUN cd /go-anduschain && rm go.sum && make clean && go mod tidy && GO111MODULE=ON && go install -buildvcs=false -v ./cmd/...
#RUN cd /go-anduschain && go run build/ci.go install

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go/bin/* /usr/local/bin/

EXPOSE 8545 8546 50505 50505/udp
ENTRYPOINT ["godaon"]