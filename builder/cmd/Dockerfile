FROM alpine:3.5

RUN apk update
# Install some utility libraries for builders extending image
RUN apk add coreutils binutils findutils grep

ADD builder /builder

EXPOSE 8001