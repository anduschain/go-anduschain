FROM ubuntu:xenial-20191010

ENV PATH=/usr/lib/go/bin:$PATH

RUN \
  apt-get update && apt-get upgrade -q -y && \
  apt-get install -y --no-install-recommends golang git make gcc libc-dev ca-certificates && \
  git clone --depth 1 https://github.com/anduschain/go-anduschain && \
  (cd go-anduschain && make godaon) && \
  cp go-anduschain/build/bin/godaon /godaon && \
  apt-get remove -y golang git make gcc libc-dev && apt autoremove -y && apt-get clean && \
  rm -rf /go-anduschain

EXPOSE 8545
EXPOSE 50505

ENTRYPOINT ["/godaon"]
