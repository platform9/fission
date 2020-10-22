# Fission Environments Redesign

As Fission supports more languages and reaches a wider set of use
cases, it's time to ask how well the current Environments design is
holding up.


## Environments V1: What we learned

Environments V1 is very simple idea: an environments is one Docker
image with an HTTP server + dynamic loader for that language; it's run
in a pod with a language-agnostic sidecar (fetcher) that downloads and
saves the function into a volume shared with the language-specific
container.

### Pros:

* Abstracted away images.

* Very fast cold start

* No image registry to manage (neither for the user nor for fission
  implementation)

* Relatively small amount of language specific code. (python env is <
  100 lines)

### Cons:

* Doesn't work well for compiled languages

* Users have to rebuild the image to add dependencies

* Only one file supported

* Errors in loading are not surfaced properly.  It is especially
  annoying to wait until runtime to see a syntax error that could have
  been caught on function upload.

* Starting a Pod without knowing the functions has its limitations: we
  can't set CPU/memory limits, we can't mount volumes (persistent
  volumes, secrets, configmaps).  We also can't change the namespace
  the Pod is in.

* Not great for a large code base

* Some people want to operate at the image level but still get the
  on-demand execution semantics of FaaS.  This is a cost-optimization
  use case.


### Discussion

Early feedback shows that almost evey user ends up rebuilding images
to add some dependencies.  Some sort of automated dependency resolution
would be very nice to have and improve the development workflow.  In
other words, just attach a package.json(nodejs) or
requirements.txt(python) with a function, and fission will do the
rest.  There's also the possibility of supporting buildpacks (simple
zipfiles), a la AWS Lambda.

Though we can support compiled languages by doing the compilation
inside the cold-start, that's not a great solution because: (a)
compile errors would be reported at runtime, and (b) because the
overhead of compilation doesn't really need to be inside the
cold-start latency.

Non-trivial functions will need multiple files.  That also helps for
common code across functions.  So we need a way for the user to define
a function as a collection of code with an entry point.

Finally, Docker images remain the most flexible way to package an app.
Today, users can always rebuild an environment image to include
anything they want.  But those images must still run a server that
implements fission-environment interface (i.e. the specialize
endpoint).  So perhaps there could be a way for users to say "don't
use environments, I've already packaged up my function, here it is".


## Environment V2 Requirements

Roughly in order of priority:

0. Retain the simplicity of the simple use cases.  First user
   experience should remain trivial -- write a function, map a URL,
   done.

1. Support compiled languages. Support error reporting on function
   upload rather than cold start.

2. Support functions as a collection of files rather than just one
   file.

3. Support automated environment-specific dependency resolution.

   (#3 may end up having the same solution as #1.  You could think of
   gathering deps as a "compilation" of package.json,
   requirements.txt, etc.)

4. Support functions as images.


### User stories

#### Environment Creation

V1 Environments were just an image.  V2 Environments will be a yaml
file with the following properties:

* Run time image (required)
* Version (required)
* Builder image (optional)
* Build invocation command (required if builder image specified)
* File name extension(s) (optional)

The version will be used to distinguish V2 environments from V1.

```
$ cat golang.yaml
type: Environment
metadata:
  name: go
spec:
  runtimeImage: fission/go-runtime
  builderImage: fission/go-builder
  buildCommand:
  - "/build.sh"
  fileExtentions:
  - go

$ fission env create -f golang.yaml
```

#### Function creation for compiled languages

User writes a function in a compiled language, for example Go.

```
$ fission function create --code blah.go

<compilation errors>

<user edits file>
$ $EDITOR blah.go
<fixes errors>

$ fission function update --code blah.go
<success>

$ fission route ... # routes work as usual
```

This same user story applies to interpreted languages too, where the
"compilation" step can be used to check for syntax errors.

#### Compiled language, without using fission builds

User compiles their function locally, resulting in a set of one or
more binaries. The user packages these up as a zip file, creating a
"deployment package".

```
$ fission function create --deployment-package foo.zip

$ fission route ... # routes work as usual
```

In this use case, fission is no longer operating at the source
level. Builds are left to the user and fission only sees the
deployment package package.

#### Collections of source files

The user can create a source package -- a set of source files in a
zip.

```
$ fission function create --source-package foo.zip

```

This workflow works similarly to providing a single source file.

In addition, fission CLI could support automatic creation of source packages, e.g.

```
$ fission function create --source-files *.js
```

This is purely client-side "syntactic sugar" -- the CLI creates the
source package instead of the user having to do it manually.  It
doesn't change semantics; the source package is still handled as one
object.

#### Handling Dependencies

The source package of a function can contain dependency specs.

Fission framework proper does not treat this spec in any special way;
it's just another file in the source package.  These will be
interpreted by the environment builder.

```
$ fission function create --source-files *.js --source-files package.json
```

In this case, the CLI will create a source package containing the JS
files and package.json.  The NodeJS environment builder will create a
deployment package out of these files.  The runtime environment will
load and run the deployment package.

#### V1 Compatibility

V1 Environments will continue to be supported.  Existing commands will
continue to work.  V1 environments won't support newer features like
builds, source and deployment packages, etc.

### Implementation

#### Environment Type

The environment type has a set of new properties: version, runtime
image, builder image, build command, file extension(s).

#### Function Type

The function type has new properties: source package, deployment
package.  The literal code string continues to be supported, but it
will have a specified size limit, say 512KB.

#### Source and Binary Packages

A package is just a zip file.  It's contents are opaque to fission:
the meaning of its contents is defined by the environment.  Fission's
job is to manage the storage and delivery of the package into build
and runtime environments.

#### Storage Service

The storage service will have an HTTP API to upload and download
files. It can store the packages on a persistent volume or as objects
in cloud storage services such as S3.

Storage service has a garbage collection API endpoint.  When invoked,
it will remove all packages that are not referenced from any function.

#### Fetcher

Fetcher gets some new responsibilities:

1. It must now also handle zip/unzip of packages

2. It must know how to upload to the storage service (so it's not
exactly "fetcher" any more, but...)

#### Runtime Environment Interface

The V2 runtime environment interface is very similar to V1
environments.  Environments must support a dynamic loader and have an
HTTP server that forwards requests to the loaded module.

The differences:

* V2 runtimes must support loading a deployment package.  Fetcher is
  responsible for unzipping a deployment package, but interpretation
  of the contents is up to the environment's code.  For example, it
  may have to include the directory where the deployment package is
  unzipped in its module load path.

[TODO any other differences?]

#### Buildmgr

A new service that will manage builds.  Its design is similar to
poolmgr, except it is triggered on creation or update of a function,
rather than HTTP requests.

Buildmgr creates a builder deployment+service for each environment.
Pods in this deployment run the environment's build container, and
fetcher, with a shared volume between the two containers.

When a function is created or updated with a source package, buildmgr
notices this and triggers a build.  First, it calls fetcher to
download the source package into a shared volume with the build
container.  It then invokes the builder by running the build
invocation command in the build container.  Next, it calls fetcher to
package up the output of the builder and store the built package into
the the storage service.

Finally, it updates the function object in the controller API with a
reference to the built package.

[We can collapse this workflow into one request into the builder
service, which would make it easier to scale up the builder
deployment; if we used multiple requests we'd need some sort of
affinity rule, but k8s services only support IP based affinity.]

#### Poolmgr

Poolmgr remains relatively unchanged.  Instead of constructing URLs for
function metadata, it uses the deployment package URL in the function
object.

#### CLI

Client libraries and CLI have to deal with the new properties in
functions and environments.

The CLI will now talk to both storage service and controller.  When a
function is created, the user can specify the function in one of 3
ways:

1. One source file, same as v1.

2. A source package (or a set of source files, which is turned into a
   source package by the CLI)

3. A deployment package

If the file is specified as a source file, fission CLI will use the
code literal if it's under the size limit; otherwise it should use the
storage service.  This will allow users to use fission deployments
with no storage service, but with a size limit on functions.

For the case of a source or deployment package, the CLI first does an
upload to the storage service, then creates a function object with a
reference to the uploaded package.
