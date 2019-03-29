ARG GO_VERSION=1.11.4

FROM golang:${GO_VERSION} AS builder

ENV GOPATH /usr
ENV APP	   ${GOPATH}/src/github.com/fission/fission/environments/go

ADD context	    ${APP}/context
ADD server.go   ${APP}

WORKDIR ${APP}
RUN go get
RUN go build -a -o /server server.go

FROM ubuntu:18.04
WORKDIR /
COPY --from=builder /server /
RUN apt update && apt install -y ca-certificates && rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["/server"]
EXPOSE 8888
