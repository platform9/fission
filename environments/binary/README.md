# Binary Environment Examples

The `binary` runtime is a go server that uses a subprocess to invoke executables or execute shell scripts.

Use Cases
- Execute bash scripts
- Execute arbitrary binaries (such as common sysadmin tools)
- Get support in _any_ programming language by executing the generated executable.


⚠️ **Words of Caution** ⚠️

The environment runs on an alpine image with some additional utility command line tools installed, such as 'grep'. 
However, in case you want to make use of more esoteric command line tools, you should add the relevant apk to the 
Dockerfile and build a new binary environment. See 'Compiling' for instructions.

When executing functions using binaries, **ensure that the executable is built for the right architecture**. 
Using the default binary environment this means that the binary should be build for Linux.

Looking for ready-to-run examples? See the [binary examples directory](../../examples/binary).

## Usage
To get started with the latest binary environment:

```bash
fission env create --name binary --image fission/binary-env --builder fission/binary-builder
```

The interface to the executable used by this environment is somewhat similar to a [CGI interface](https://en.wikipedia.org/wiki/Common_Gateway_Interface).
This means that any HTTP headers are converted to environment variables of the form "HTTP_<header-name>". For example these
are some of frequently occurring headers:

```bash
# Request Metadata
CONTENT_LENGTH
REQUEST_URI
REQUEST_METHOD

# HTTP Headers
HTTP_ACCEPT
HTTP_USER-AGENT
HTTP_CONTENT-TYPE
# ...
```

The body of HTTP piped over the STDIN to the executable. 
All output that is provided to the server over the STDOUT will be transformed into the HTTP response.

## Compiling

To build the runtime environment:
```bash
docker build --tag=${USER}/binary-env .
```

To build the builder environment:
```bash
(cd builder/ && docker build --tag=${USER}/binary-builder .)
```
