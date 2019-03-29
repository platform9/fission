FROM ruby:2.6.1-alpine3.9

RUN apk update
RUN apk add --no-cache build-base

COPY . /app
WORKDIR /app
RUN bundle install

ENTRYPOINT ["ruby"]
CMD ["server.rb"]
