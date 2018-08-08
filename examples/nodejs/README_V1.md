# Fission Node.js Examples

This directory contains several examples to get you started using Node.js with Fission.

## Environment

Before running any of these functions, make sure you have created a `nodejs` Fission environment:

```
$ fission env create --name nodejs --image fission/node-env
```

Note: The default `fission/node-env` image is based on Alpine, which is much smaller than the main Debian Node image (65MB vs 680MB) while still being suitable for most use cases.
If you need to use the full Debian image use the `fission/node-env-debian` image instead.
See the [official Node docker hub repo](https://hub.docker.com/_/node/) for considerations
relating to this choice.

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

Since it is an `async` function, you can `await` `Promise`s, as demonstrated in the `weather.js` function.

## hello.js

This is a basic "Hello, World!" example. It simply returns a status of `200` and text body.

### Usage

```bash
# Upload your function code to fission
$ fission function create --name hello --env nodejs --code hello.js

# Map GET /hello to your new function
$ fission route create --method GET --url /hello --function hello

# Run the function.
$ curl http://$FISSION_ROUTER/hello
Hello, world!
```

## hello-callback.js

This is a basic "Hello, World!" example implemented with the legacy callback implementation. If you declare your function with two arguments (`context`, `callback`), a callback taking three arguments (`status`, `body`, `headers`) is provided.

⚠️️ Callback support is only provided for backwards compatibility! We recommend that you use `async` functions instead.

### Usage

```bash
# Upload your function code to fission
$ fission function create --name hello-callback --env nodejs --code hello-callback.js

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
$ fission fn create --name kubeEventsSlack --env nodejs --code kubeEventsSlack.js

# Watch all services in the default namespace:
$ fission watch create --function kubeEventsSlack --type service --ns default
```

## weather.js

In this example, the Yahoo Weather API is used to current weather at a given location.

### Usage

```bash
# Upload your function code to fission
$ fission function create --name weather --env nodejs --code weather.js

# Map GET /stock to your new function
$ fission route create --method POST --url /weather --function weather

# Run the function.
$ curl -H "Content-Type: application/json" -X POST -d '{"location":"Sieteiglesias, Spain"}' http://$FISSION_ROUTER/weather

{"text":"It is 2 celsius degrees in Sieteiglesias, Spain and Mostly Clear"}
```
