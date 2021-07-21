FROM golang:1
ENV GO111MODULE on
RUN mkdir -p /go/src/github.com/fission
RUN wget -qO- https://download.docker.com/linux/static/stable/x86_64/docker-17.03.0-ce.tgz | tar xvz -C /usr/local/bin/ --strip 1
RUN wget -qO- https://get.helm.sh/helm-v3.3.0-linux-amd64.tar.gz | tar xvz -C /usr/local/bin/ --strip 1
