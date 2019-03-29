# Ruby examples

This directory contains several examples to get you started using Ruby
with Fission.

Before running any of these functions, make sure you have created a
`ruby` Fission environment:

```
$ fission env create --name ruby --image USER/ruby-env
```

## Method signature

A standard Ruby function has the basic form:

```ruby
def handler(context)
  return [200, {}, []]
end
```

If the fission context is not required, the function can be simplified:

```ruby
def handler
  [200, {}, ["Hello, world!\n"]]
end
```

If a simple text response is to be returned, with a status of 200, this
can be further simplified.

```ruby
def handler
  "Hello, world!\n"
end
```

## Hello example (`hello.rb`)

This example is the simplest possible Ruby function, as described above.

To run the example:

```
$ fission function create --name hello --env ruby --code examples/ruby/hello.rb

$ fission route create --method GET --url /hello --function hello

$ curl http://$FISSION_ROUTER/hello
  Hello, world!
```

## Request data example (`request_data.rb`)

This example shows basic use of the `Fission::Context` and
`Fission::Request` objects.

To run the example:

```
$ fission function create --name request --env ruby --code examples/ruby/request_data.rb

$ fission route create --method GET --url /request/{id} --function request

$ curl http://$FISSION_ROUTER/request/123?key=abc
  ---ENV---
  GATEWAY_INTERFACE=CGI/1.1
  PATH_INFO=/
  QUERY_STRING=key=abc
  REMOTE_ADDR=172.17.0.8
  REMOTE_HOST=172.17.0.8
  REQUEST_METHOD=GET
  REQUEST_URI=http://192.168.64.200:31314/?key=abc
  SCRIPT_NAME=
  SERVER_NAME=192.168.64.200
  SERVER_PORT=31314
  SERVER_PROTOCOL=HTTP/1.1
  SERVER_SOFTWARE=WEBrick/1.3.1 (Ruby/2.4.1/2017-03-22)
  HTTP_HOST=192.168.64.200:31314
  HTTP_USER_AGENT=curl/7.52.1
  HTTP_ACCEPT=*/*
  HTTP_X_FISSION_PARAMS_ID=123
  HTTP_X_FORWARDED_FOR=172.17.0.1
  HTTP_ACCEPT_ENCODING=gzip
  rack.version=1=3
  ...
  HTTP_VERSION=HTTP/1.1
  REQUEST_PATH=/

  ---HEADERS---
  Accept: */*
  Accept-Encoding: gzip
  Host: 192.168.64.200:31314
  User-Agent: curl/7.52.1
  Version: HTTP/1.1
  X-Fission-Params-Id: 123
  X-Forwarded-For: 172.17.0.1

  ---PARAMS---
  key=abc
  id=123

  --BODY--

```

## V2 Specification Example (with builder support)

```
$ fission function create --name parse --env ruby --src "parse/*" --entrypoint handler 
$ fission fn test --name parse --body '<message>This is my message</message>'
  This is my message
```
