# Fission: Go Environment

This is the Go environment for Fission.

It's a Docker image containing a Go runtime, along with a dynamic loader.

## Build this image

```
docker build -t USER/go-runtime . && docker push USER/go-runtime
```

Note that if you build the runtime, you must also build the go-builder
image, to ensure that it's at the same version of go:

```
cd builder && docker build -t USER/go-builder . && docker push USER/go-builder
```

## Using the image in fission

You can add this customized image to fission with "fission env
create":

```
fission env create --name go --image USER/go-runtime --builder USER/go-builder --version 2
```

Or, if you already have an environment, you can update its image:

```
fission env update --name go --image USER/go-runtime --builder USER/go-builder
```

After this, fission functions that have the env parameter set to the
same environment name as this command will use this environment.

## Creating functions to use this image

See the [examples README](examples/go/README.md).
