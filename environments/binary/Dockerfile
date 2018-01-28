FROM golang:onbuild

WORKDIR /go
COPY *.go /go/

RUN GOOS=linux GOARCH=386 go build -o server .

FROM alpine:3.5

WORKDIR /app

RUN apk update
RUN apk add coreutils binutils findutils grep

COPY --from=0 /go/server /app/server

EXPOSE 8888
ENTRYPOINT ["./server"]
