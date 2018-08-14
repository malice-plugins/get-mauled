FROM malice/alpine

LABEL maintainer "https://github.com/blacktop"

RUN apk --update add --no-cache ca-certificates p7zip

COPY . /go/src/github.com/maliceio/getmauled
RUN apk --update add --no-cache -t .build-deps \
    openssl-dev \
    jansson-dev \
    build-base \
    mercurial \
    musl-dev \
    openssl \
    bash \
    wget \
    git \
    gcc \
    dep \
    go \
    && echo "===> Building getmauled Go binary..." \
    && cd /go/src/github.com/maliceio/getmauled \
    && export GOPATH=/go \
    && go version \
    && dep ensure \
    && CGO_ENABLED=1 go build -ldflags "-s -w -X main.Version=$(cat VERSION) -X main.BuildTime=$(date -u +%Y%m%d)" -o /bin/getmauled \
    && rm -rf /go /usr/local/go /usr/lib/go /tmp/* \
    && apk del --purge .build-deps

WORKDIR /malware

ENTRYPOINT ["su-exec","malice","getmauled"]
CMD ["--help"]