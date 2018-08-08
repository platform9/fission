# Fission Node.js Examples

This is V2 example, check [here](README_V1.md) for V1.

This directory contains several examples to get you started using Node.js with Fission.

Before running any of these functions, make sure you have created a `nodejs` Fission environment:

```bash
# Create an environment with default nodejs images
$ fission env create --name nodeenv --image fission/node-env:latest --builder fission/node-builder:latest
# Create zip file from our example
$ zip -jr nodejs.zip nodejs/
# Create a package with the zip file
$ fission pkg create --sourcearchive nodejs.zip --env nodeenv
```

## Function signature

Every Node.js function has the same basic form:

```javascript
module.exports = async function(context) {
    return {
        status: 200,
        body: 'Your body here',
        headers: {
            'Foo': 'Bar'
        }
    }
}
```
## hello.js

This is a basic "Hello, World!" example. It simply returns a status of `200` and text body.

### Usage
Since it is an `async` function, you can `await` `Promise`s, as demonstrated in the `weather.js` function.

```bash
# Create a function
$ fission fn create --name hello --pkg [pkgname] --entrypoint "hello"

# Test the function
$ fission fn test --name hello
```

## index.js

This file does nothing but for demonstrating `require` feature.

### Usage
```bash
# Create a function, you can skip `--entrypoint` as node will look for `index.js` by default
$ fission fn create --name index --pkg [pkgname]

# Test the function
$ fission fn test --name index
```

## multi-entry.js

This is a multiple exports example. There are two exports: entry1 and entry2

### Usage
```bash
# Create a function for entry1
$ fission fn create --name entry1 --pkg [pkgname]  --entrypoint "multi-entry.entry1"

# Test the function
$ fission fn test --name entry1

# Create a function for entry2
$ fission fn create --name entry2 --pkg [pkgname]  --entrypoint "multi-entry.entry2"

# Test the function
$ fission fn test --name entry2
```

## hello-callback.js

This is a basic "Hello, World!" example implemented with the legacy callback implementation. If you declare your function with two arguments (`context`, `callback`), a callback taking three arguments (`status`, `body`, `headers`) is provided.

⚠️️ Callback support is only provided for backwards compatibility! We recommend that you use `async` functions instead.

### Usage

```bash
# Create a function
$ fission fn create --name hello-callback --pkg [pkgname] --entrypoint "hello-callback"

# Map GET /hello-callback to your new function
$ fission route create --method GET --url /hello-callback --function hello-callback

# Run the function.
$ curl http://$FISSION_ROUTER/hello-callback
Hello, world!
```

## kubeEventsSlack.js

This example watches Kubernetes events and sends them to a Slack channel. To use this, create an incoming webhook for your Slack channel, and replace the `slackWebhookPath` in the example code.

### Usage

```bash
# Upload your function code to fission
$ fission fn create --name kubeEventsSlack --pkg [pkgname] --entrypoint "hello-callback"

# Watch all services in the default namespace:
$ fission watch create --function kubeEventsSlack --type service --ns default
```

## weather.js

In this example, the Yahoo Weather API is used to current weather at a given location.

### Usage

```bash
# Upload your function code to fission
$ fission function create --name weather --pkg [pkgname] --entrypoint "weather"

# Map GET /stock to your new function
$ fission route create --method POST --url /weather --function weather

# Run the function.
$ curl -H "Content-Type: application/json" -X POST -d '{"location":"Sieteiglesias, Spain"}' http://$FISSION_ROUTER/weather

{"text":"It is 2 celsius degrees in Sieteiglesias, Spain and Mostly Clear"}
```
