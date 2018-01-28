# Binary Environment Examples

The `binary` runtime is a go server that uses a subprocess to invoke executables or execute shell scripts.

Use Cases
- Execute bash scripts
- Run executables of languages that have no dedicated environment yet.

## Words of Caution
The environment runs on an alpine image with some additional utilty commandline tools installed, such as 'grep'. 
However, in case you want to make use of more ecsoteric commandline tools, you should add the relevant apk to the 
Dockerfile and build a new binary environment. See 'Compiling' for instructions.

When executing functions using binaries, **ensure that the executable is built for the right architecture**. 
Using the default binary environment this means that the binary should be build for Linux.

## Usage
The interface to the executable used by this environment is somewhat similar to a [CGI interface](https://en.wikipedia.org/wiki/Common_Gateway_Interface).
This means that any HTTP headers are converted to environment variables of the form "HTTP_<header-name>". For example these
are some of frequently occurring headers:

```
# Request Metadata
CONTENT_LENGTH
REQUEST_URI
REQUEST_METHOD

# HTTP Headers
HTTP_ACCEPT
HTTP_USER-AGENT
...
```

The body of HTTP piped over the STDIN to the executable. 
All output that is provided to the server over the STDOUT will be transformed into the HTTP response.


## Compiling

Build the Dockerfile:
```bash
docker build --tag=${USER}/binary-env .
```

See the [README](../../examples/binary/README.md) in the binary examples directory for usage instructions.
