'use strict';

//
// Watch kubernetes events and send them to a slack channel.  This
// uses Slack's incoming webhooks.  To use this, create an incoming
// webhook for your slack channel through Slack's UI, and populate the
// relative path below.
//
// Create the function in fission:
//
//   fission fn create --name kubeEventsSlack --env nodejs --code kubeEventsSlack.js
//
// Then, watch all services in the default namespace:
// 
//   fission watch create --function kubeEventsSlack --type service --ns default
//

let https = require('https');

const slackWebhookPath = "YOUR RELATIVE PATH HERE"; // Something like "/services/XXX/YYY/zZz123"

const upcaseFirst = (s) => {
    return s.charAt(0).toUpperCase() + s.slice(1).toLowerCase();
}

const sendSlackMessage = async (msg) => {
    const postData = `{"text": "${msg}"}`;
    const options = {
        hostname: "hooks.slack.com",
        path: slackWebhookPath,
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        }
    };

    return new Promise(function (resolve, reject) {
        const req = https.request(options, function (res) {
            console.log(`slack request status = ${res.statusCode}`);
            return resolve();
        });
        req.send(postData);
    });
}

module.exports = async (context) => {
    const body = context.request.body;
    const version = body.metadata.resourceVersion;
    const eventType = context.request.get('X-Kubernetes-Event-Type');
    const objType = context.request.get('X-Kubernetes-Object-Type');

    const msg = `${upcaseFirst(eventType)} ${objType} ${body.metadata.name}`;
    console.log(msg, version);

    if (eventType == 'DELETED' || eventType == 'ADDED') {
        console.log("sending event to slack")
        await sendSlackMessage(msg);
    }

    return {
        status: 200,
        body: ""
    }
}
