<p align="center">
  <img src="https://docs.fission.io/images/logo.png" width="300" />
  <br>
  <h1 align="center">Fission: Serverless Functions for Kubernetes</h1>
</p>
<p align="center">
  <a href="https://github.com/fission/fission/actions?query=workflow%3A%22Fission+CI%22">
    <img src="https://github.com/fission/fission/workflows/Fission%20CI/badge.svg?branch=master" alt="Build Status" />
  </a>
  
  <a href="https://goreportcard.com/report/github.com/fission/fission">
    <img src="https://goreportcard.com/badge/github.com/fission/fission" alt="Go Report Card" />
  </a>
  <a href="https://codecov.io/gh/fission/fission">
    <img src="https://codecov.io/gh/fission/fission/branch/master/graph/badge.svg" alt="codecov" />
  </a>
  <br>
  <a href="http://fission.io">fission.io</a> | <a href="http://twitter.com/fissionio">@fissionio</a> | <a href="https://join.slack.com/t/fissionio/shared_invite/enQtOTI3NjgyMjE5NzE3LTllODJiODBmYTBiYWUwMWQxZWRhNDhiZDMyN2EyNjAzMTFiYjE2Nzc1NzE0MTU4ZTg2MzVjMDQ1NWY3MGJhZmE">Slack</a>
</p>

--------------

Fission is a fast serverless framework for Kubernetes with a focus on
developer productivity and high performance.

Fission operates on _just the code_: Docker and Kubernetes are
abstracted away under normal operation, though you can use both to
extend Fission if you want to.

Fission is extensible to any language; the core is written in Go, and
language-specific parts are isolated in something called
_environments_ (more below).  Fission currently supports NodeJS, Python, Ruby, Go, 
PHP, Bash, and any Linux executable, with more languages coming soon.

Table of Contents
=================

   * [Fission: Serverless Functions for Kubernetes](#fission-serverless-functions-for-kubernetes)
      * [Performance: 100msec cold start](#performance-100msec-cold-start)
      * [Kubernetes is the right place for Serverless](#kubernetes-is-the-right-place-for-serverless)
      * [Getting Started](#getting-started)
      * [Learn More](#learn-more)
      * [Contributing](#contributing)
      * [Get Help &amp; Community Meeting](#get-help--community-meeting)
      * [Official Releases](#official-releases)
      * [Sponsors](#sponsors)
   * [Licensing](#licensing)

## Performance: 100msec cold start

Fission maintains a pool of "warm" containers that each contain a
small dynamic loader.  When a function is first called,
i.e. "cold-started", a running container is chosen and the function is
loaded.  This pool is what makes Fission fast: cold-start latencies
are typically about 100msec.

## Kubernetes is the right place for Serverless

We're built on Kubernetes because we think any non-trivial app will
use a combination of serverless functions and more conventional
microservices, and Kubernetes is a great framework to bring these
together seamlessly.

Building on Kubernetes also means that anything you do for operations
on your Kubernetes cluster &mdash; such as monitoring or log
aggregation &mdash; also helps with ops on your Fission deployment.

## Getting Started

```bash
  # Add the stock NodeJS env to your Fission deployment
  $ fission env create --name nodejs --image fission/node-env

  # Create a function with a javascript one-liner that prints "hello world"
  $ fission function create --name hello --env nodejs --code https://raw.githubusercontent.com/fission/fission/master/examples/nodejs/hello.js

  # Run the function.  This takes about 100msec the first time.
  $ fission function test --name hello
  Hello, world!
```

## Learn More

* Understand [Fission Concepts](https://docs.fission.io/docs/concepts/).
* See the [installation guide](https://docs.fission.io/docs/installation/) for installing and running Fission.
* You can learn more about Fission and get started from [Fission Docs](https://docs.fission.io/docs).
* See the [troubleshooting guide](https://docs.fission.io/docs/trouble-shooting/) for debugging your functions and Fission installation.

## Contributing

Check out the [contributing guide](CONTRIBUTING.md).

## Get Help & Community Meeting 

Fission is a project by [many contributors](https://github.com/fission/fission/graphs/contributors).
Reach us on [slack](https://join.slack.com/t/fissionio/shared_invite/enQtOTI3NjgyMjE5NzE3LTllODJiODBmYTBiYWUwMWQxZWRhNDhiZDMyN2EyNjAzMTFiYjE2Nzc1NzE0MTU4ZTg2MzVjMDQ1NWY3MGJhZmE) or [twitter](https://twitter.com/fissionio).

A regular community meeting takes place every other Thursday at 09:00 AM PT (Pacific Time). [Convert to your local timezone](http://www.thetimezoneconverter.com/?t=09:00&tz=PT%20%28Pacific%20Time%29).

Meeting Link: https://zoom.us/j/413921817 

The meeting agenda for next meeting and notes from past meetings are maintained in [this document](https://docs.google.com/document/d/1E-xw4KJgka4sUpETHxr9BJBYntzrtxlAN_CE3Wt8kws). You are welcome to join to discuss direction of project, design and implementation reviews and general questions about project etc.

## Official Releases

Official releases of Fission can be found on [the releases page](https://github.com/fission/fission/releases). 
Please note that it is strongly recommended that you use official releases of Fission, as unreleased versions from 
the master branch are subject to changes and incompatibilities that will not be supported in the official releases. 

## Sponsors

The following companies, organizations, and individuals support Fission's ongoing maintenance and development. If you are using/contributing to Fission, we would be happy to list you here, please raise a Pull request.

<p>
    <a href="https://infracloud.io/"><img src="https://fission.io/sponsors/infracloud.png" alt="InfraCloud" height="70"></a>
    <a href="https://srcmesh.com/"><img src="https://fission.io/sponsors/srcmesh.png" alt="Srcmesh" height="70"></a>
    <a href="https://www.digitalocean.com/?utm_medium=opensource&utm_source=fissionio">
      <img src="https://opensource.nyc3.cdn.digitaloceanspaces.com/attribution/assets/PoweredByDO/DO_Powered_by_Badge_blue.svg" width="201px">
    </a>
</p>

# Licensing

Fission is under the Apache 2.0 license.
