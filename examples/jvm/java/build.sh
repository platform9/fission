#!/bin/sh
# This script allows you to build the jar without needing Maven & JDK installed locally. 
# You need docker, as it uses a Docker image to build source code
set -eou pipefail
docker run -it --rm  -v "$(pwd)":/usr/src/mymaven -w /usr/src/mymaven maven:3.5-jdk-8 mvn clean package
