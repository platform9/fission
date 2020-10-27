# Java Environment : Design & considerations

This document documents the design and thoughts that lead to design of Java environment. Before we dive deeper, some important points:

- When we say Java, we really mean JVM. That does not mean that all languages will work seamlessly, so support will be added gradually based on validation. Some of popular languages as of today are:
    - Scala
    - Groovy
    - Kotlin (Server side with Spring)

- In Java there are a few prominent frameworks which have a ecosystem of their own (See list below). How these framework fit in the environment will be detailed later, but it is important to understand their place in ecosystem and design for it.
    - A large percentage of enterprise developers use [Spring framework](https://spring.io/) as has been shown by multiple surveys
    - Reactive has taken up recently with data intensive operations [Reactive extensions for JVM](https://github.com/ReactiveX/RxJava)
    - [Spark](http://sparkjava.com/) is a micro web framework

A draft implementation of the Java environment design is in the branch [java_env_alpha](https://github.com/fission/fission/tree/java_env_alpha). Also a earlier implementation based on Vort.x framework [can be found here](https://github.com/tobias/fission-java-env/)

## Function interface

The goal is here is to minimize the lock in for the user into any framework as much as possible. Java 8 introduced an interface called ```Function``` which could be a great fit here. The user has to implement the ```Function<T, R>``` class and to meet the contract implement the apply method:

```
public class HelloWorld implements Function<T, R> {

	public R apply(T str) {

```

Now - the T & R could be different things and we discuss some options below:

### Body in request and response

From the early implementation in the branch mentioned above, the environment extracts the body and send it as a JSON string. The JSON then can be transformed into the appropriate object by the function.

This works well, but has one major limitation: the function does not get access to other things like headers etc. The same thing applies to the response: function can send the body but looses control over status code etc.

### HttpServletRequest and HttpServletResponse

It is possible to send the [HttpServeletRequest](https://docs.oracle.com/javaee/6/api/javax/servlet/http/HttpServletRequest.html) request object as it is to the function class but then the interface becomes a bit too low level. For example the function user has to retrieve the body of request using ```getInputStream``` which gives raw input stream and needs additional work. 

Also most enterprise applications today use a framework of some sort for web applications instead of dealing with the raw HttpServlet

### Custom/Context Object

A custom object which encapsulates all needed fields etc. can be used to pass the data from environment to the function. But this means the user has to import a Fission object/library for this object in the application code.

This approach has been taken in the implementation done earlier for a Java environment in Fission and [object interface can be found here](https://github.com/tobias/fission-java-env/blob/master/src/main/java/io/fission/api/Context.java). Related discussion is in the [issue](https://github.com/fission/fission/issues/91)

AWS Lambda also uses a context object, but the purpose is very different, [details of context object here](https://docs.aws.amazon.com/lambda/latest/dg/java-context-object.html).

### Spring's HttpEntity

If we have to depend on a class/library, it is probably better to depend on a class which is part of ecosystem. So instead of using the low level interface of Servlet, we can use [HttpEntity's subclasses RequestEntity and ResponseEntity](https://docs.spring.io/spring/docs/5.0.5.RELEASE/javadoc-api/org/springframework/http/HttpEntity.html). This ensures that the function user is not locked in the Fission object contract, but also gets the full access to request/response object.

The Spring cloud function project also discusses the issue of not having access to other things in request and [related issues are here](https://github.com/spring-cloud/spring-cloud-function/issues?utf8=%E2%9C%93&q=is%3Aissue+is%3Aopen+header)

### Thoughts

- If we only intend to pass request/response object to function - then using HttpEntity might be a good choice

- If there is a need for additional exchange of information between the environment and function execution in future, then a custom/context object is a better option. We can wrap the HttpEntity's fields and additional fields in the custom context object

## Environment Design

JVM environment design is based on Spring boot and Spring MVC frameworks. The details can be found in branch, but here are some key points:

- All classes in the function and dependent classes are loaded into JVM. Which means the user should supply the uber/fat jar for execution.
- The entrypoint class is specified by the user as ```entrypoint``` flag on the class. The method is by convention (```apply``` as per the Function interface contract)
