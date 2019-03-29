# Fission: PHP Environment

This is the PHP environment for Fission.

It's a Docker image containing a PHP 7.3 runtime. This image use php:7.3-cli base image with the built-in PHP server

This environment didn't force you to use class or create a main function.

A few common extensions are included :
- curl
- gd
- json
- mcrypt
- PDO (pdo_mysql, pdo_pgsql, pdo_sqlite)
- OPcache
- ftp
- iconv
- phar
- mysqli
- pgsql
- SimpleXML
- xmlrpc
- zip

## Customizing this image

To add other extensions or packages(composer.json) you need to edit the Dockerfile and rebuild this image (instructions below).

## Rebuilding and pushing the image

You'll need access to a Docker registry to push the image: you can
sign up for Docker hub at hub.docker.com, or use registries from
gcr.io, quay.io, etc.  Let's assume you're using a docker hub account
called USER.  Build and push the image to the the registry:

```
   docker build -t USER/php7-env . && docker push USER/php7-env
```

## Using the image in fission

You can add this customized image to fission with "fission env
create":

```
   fission env create --name php7 --image USER/php7-env
```

Or, if you already have an environment, you can update its image:

```
   fission env update --name php7 --image USER/php7-env
```

After this, fission functions that have the env parmeter set to the
same environment name as this command will use this environment.
