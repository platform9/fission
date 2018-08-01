ARG BUILDER_IMAGE=fission/builder
FROM ${BUILDER_IMAGE}
# default variant is the official alpine node image (much smaller than the standard image)
FROM node:8-alpine

ARG NODE_ENV
ENV NODE_ENV $NODE_ENV

COPY --from=0 /builder /builder
ADD build.sh /usr/local/bin/build
RUN chmod +x /usr/local/bin/build
