# Commented lines are temp, till we get the Fission java lib published to Maven central
#FROM maven:3.5-jdk-8 as BUILD
#COPY src /usr/src/myapp/src
#COPY pom.xml /usr/src/myapp
#RUN mvn -f /usr/src/myapp/pom.xml clean package

FROM openjdk:8-jdk-alpine
VOLUME /tmp
#COPY --from=BUILD /usr/src/myapp/target/jvmenv-0.0.1-SNAPSHOT.jar /app.jar
COPY target/jvmenv-0.0.1-SNAPSHOT.jar /app.jar
ENTRYPOINT ["java","-Djava.security.egd=file:/dev/./urandom","-jar","/app.jar","--server.port=8888"]
EXPOSE 8888