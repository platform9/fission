# Environment V2 Fission-Environment API

Fission Environments are the language-specific component of fission.

They must satisfy the interface in this spec.

## Meta

This is version 2.0-alpha of the Fission-Environment API.

(It's unstable and may change without warning until 2.0-beta.)

## Overview

Fission V2 Environments consist of:

 * Metadata
 * A runtime image
 * A builder image (optional)
 * Function Interface Specification
 * User Documentation
 * Examples

### Metadata 

See EnvironmentSpec, Runtime, and Builder in types.go.

## Runtime Image

An environment runtime image is a docker container image. It must run
a server that has two jobs:

(a) Loading a "function" from a file path on demand
(b) Invoking that function on request

### Function Loading

The environment must expose a single HTTP endpoint (at the port and
URL specified in the metadata) that loads a function. The function
load request is a JSON-serialized `FunctionLoadRequest`.

The function load request contains a filepath to load the function
from. Fission does not define in any way the contents of the path; it
is completely environment-dependent. It may be a single file, or a
directory (in case of deployment packages).

The load request may contain an EntryPoint. If it does, the loader
must interpret this; usually it's the name of a function in a module
or package containing multiple functions. If there is no entry point,
the environment must use a default; again, the value of this default
is environment-specific.

The load request may contain a URL. If it does, requests to that URL
should be routed to the function. It defaults to "/".

### Function Invocation

Functions are invoked on HTTP request to the server. The port for the
request on the runtime container is defined in the Runtime metadata,
and the URL for the request is specified in the FunctionLoadRequest.

The interface of the function is environment specific; the environment
must come with a spec for this interface.

## Builder

The builder is a container image that contains tools to build a
function from source. The source may be a single file or a directory
of files.

The builder container is invoked with the specified command, with the
following params:

 1. File path of the source
 2. File path where the output should go
 3. Other env or function-specific params passed by the user

The first two parameters are file paths, and all remaining params are
environment-specific.

The output of the builder should be something that the runtime can
load and run -- there should be no intermediate steps that need user
intervention.

### Errors

## Function Interface Spec

The function interface spec is a document that specifies the interface
of functions and their semantics.  It must specify:

 * How functions are invoked (sync, async)
 * How the request context is provided to the function (URL, headers, request type, request body)
 * Function logging
 * Semantics of function errors and exceptions


## Documentation

The docs should contain everything necessary to use the environment:

 * How to add it to a fission cluster
 * How to write and build functions for this environment (link to the interface spec)
 * How to modify and rebuild the environment itself

## Examples

Suggested examples to provide:

 * A simple "Hello world"
 * A function that demonstrates use of the request context: url params,
   request headers, request body 
 * A function that does logging
 * A multi-file function package
 * A function with dependencies
 * Functions with shared code

## Compatibility with v1

V1 environment images can be used as v2 environment runtime images. 
