# Fission: Java and JVM-Jersey Environment

This is the JVM (Jersey based) environment for Fission.

It's a Docker image containing a OpenJDK8 runtime, along with a
dynamic loader.  A few dependencies are included in the
pom.xml file.

Unlike the other [JVM environment](../jvm) which is based on the Spring framework, this environment uses Jersey.

Looking for ready-to-run examples? See the [JVM examples directory](../../examples/jvm-jersey).

## Customizing this image

To add package dependencies, edit pom.xml to add what you
need, and rebuild this image (instructions below).

## Rebuilding and pushing the image

You'll need access to a Docker registry to push the image: you can
sign up for Docker hub at hub.docker.com, or use registries from
gcr.io, quay.io, etc.  Let's assume you're using a docker hub account
called USER.  Build and push the image to the the registry:

```
   docker build -t USER/jvm-env-jersey . && docker push USER/jvm-env-jersey
```

## Using the image in fission

You can add this customized image to fission with "fission env
create":

```
   fission env create --name jvm --image USER/jvm-env-jersey
```

Or, if you already have an environment, you can update its image:

```
   fission env update --name jvm --image USER/jvm-env-jersey 
```

After this, fission functions that have the env parameter set to the
same environment name as this command will use this environment.

## Web Server Framework

JVM Jersey environment uses an embedded Jetty HTTP server by default, as can be seen in the pom.xml file.

```
<dependency>
	<groupId>org.eclipse.jetty</groupId>
	<artifactId>jetty-server</artifactId>
	<version>9.0.4.v20130625</version>
</dependency>
<dependency>
	<groupId>org.eclipse.jetty</groupId>
	<artifactId>jetty-servlet</artifactId>
	<version>9.0.4.v20130625</version>
</dependency>
```

## Java and JVM builder

JVM environment builder is based on OpenJDK8 and Maven 3.5.4 version. The default build command runs `mvn clean package` and uses the target/*with-dependencies.jar file for function. The default build command can be overridden as long as the uber jar file is copied to ${DEPLOY_PKG}.
