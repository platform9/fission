# Programming Model

This document describes the programming model for fission functions.  

## Time Limits

By default, there is no time limit on fission functions.  

Idle running instances may be killed at any time (usually after the
default idle timeout of 10 minutes, but this is configurable).


## HTTP Triggers

Functions triggered over HTTP receive the HTTP request object in the
context.  The request's query string, POST body, etc. can be retrieved
from this object.  The interface is language-specific: see [TODO] for
documentation on the context object in each environment.


## Kubernetes Watch Event Triggers

Kubernetes watches can be used to trigger functions. These functions
receive the Kubernetes watch.Event object in JSON-serialized form.
