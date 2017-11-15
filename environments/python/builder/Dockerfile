ARG BUILDER_IMAGE=fission/builder:latest
FROM ${BUILDER_IMAGE}

RUN apk update
RUN apk add --no-cache python3 python3-dev build-base
RUN pip3 install --upgrade pip
RUN rm -r /root/.cache

ADD defaultBuildCmd /usr/local/bin/build

EXPOSE 8001
