package io.fission;

import java.io.File;
import java.io.IOException;
import java.io.PrintWriter;
import java.net.MalformedURLException;
import java.net.URL;
import java.net.URLClassLoader;
import java.util.Enumeration;
import java.util.jar.JarEntry;
import java.util.jar.JarFile;
import java.util.logging.Level;
import java.util.logging.Logger;
import javax.ws.rs.GET;
import javax.ws.rs.POST;
import javax.ws.rs.PUT;
import javax.ws.rs.DELETE;
import javax.ws.rs.Path;
import javax.ws.rs.core.Response;
import javax.ws.rs.core.Context;
import javax.ws.rs.container.ContainerRequestContext;

import io.fission.Function;

@Path("/")
public class JerseyServer {

	private static Function<ContainerRequestContext,Response> fn;

	private static final int CLASS_LENGTH = 6;

	private static Logger logger = Logger.getGlobal();

	@GET
	public Response home(@Context ContainerRequestContext request) {
		return callUserFunction(request);
	}

	@POST
	public Response homePost(@Context ContainerRequestContext request) {
		return callUserFunction(request);
	}

	@PUT
	public Response homePut(@Context ContainerRequestContext request) {
		return callUserFunction(request);
	}

	@DELETE
	public Response homeDelete(@Context ContainerRequestContext request) {
		return callUserFunction(request);
	}

	@Path("v2/specialize")
	@POST
	public Response specialize(FunctionLoadRequest req) {

		long startTime = System.nanoTime();
		File file = new File(req.getFilepath());

		if (!file.exists()) {
			return Response.status(Response.Status.BAD_REQUEST).entity("/userfunc/usernot found").build();
		}

		String entryPoint = req.getFunctionName();
		logger.log(Level.INFO, "Entrypoint class:" + entryPoint);
		if (entryPoint == null) {
			return Response.status(Response.Status.BAD_REQUEST).entity("Entrypoint class is missing in the JAR or the name is incorrect")
					.build();
		}

		JarFile jarFile = null;
		ClassLoader cl = null;
		try {

			jarFile = new JarFile(file);
			Enumeration<JarEntry> e = jarFile.entries();
			URL[] urls = { new URL("jar:file:" + file + "!/") };

			// TODO Check if the class loading can be improved for ex. use something like:
			// Thread.currentThread().setContextClassLoader(cl);
			if (this.getClass().getClassLoader() == null) {
				cl = URLClassLoader.newInstance(urls);
			} else {
				cl = URLClassLoader.newInstance(urls, this.getClass().getClassLoader());
			}

			if (cl == null) {
				return Response.status(Response.Status.BAD_REQUEST).entity("Failed to initialize the class loader")
						.build();
			}

			// Load all dependent classes from libraries etc.
			while (e.hasMoreElements()) {
				JarEntry je = e.nextElement();
				if (je.isDirectory() || !je.getName().endsWith(".class")) {
					continue;
				}
				String className = je.getName().substring(0, je.getName().length() - CLASS_LENGTH);
				className = className.replace('/', '.');
				cl.loadClass(className);
			}

			// Instantiate the function class
			fn = (Function) cl.loadClass(entryPoint).newInstance();
		} catch (MalformedURLException e) {
			e.printStackTrace();
			return Response.status(Response.Status.BAD_REQUEST).entity("Entrypoint class is missing in the function")
					.build();
		} catch (ClassNotFoundException e) {
			e.printStackTrace();
			return Response.status(Response.Status.BAD_REQUEST).entity("Error loading Function or dependent class")
					.build();
		} catch (InstantiationException e) {
			e.printStackTrace();
			return Response.status(Response.Status.BAD_REQUEST)
					.entity("Error creating a new instance of function class").build();
		} catch (IllegalAccessException e) {
			e.printStackTrace();
			return Response.status(Response.Status.BAD_REQUEST)
					.entity("Error creating a new instance of function class").build();
		} catch (IOException e) {
			e.printStackTrace();
			return Response.status(Response.Status.BAD_REQUEST).entity("Error reading the JAR file").build();
		} finally {
			try {
				jarFile.close();
			} catch (IOException e) {
				e.printStackTrace();
				return Response.status(Response.Status.BAD_REQUEST)
						.entity("Error closing the file while loading the class").build();
			}
		}
		long elapsedTime = System.nanoTime() - startTime;
		logger.log(Level.INFO, "Specialize call done in: " + elapsedTime / 1000000 + " ms");
		return Response.status(Response.Status.OK).entity("Done").build();
	}

	private Response callUserFunction(ContainerRequestContext httpRequest) {
		if (fn == null) {
			return Response.status(Response.Status.BAD_REQUEST).entity("Container not specialized").build();
		} else {
			return fn.call(httpRequest, null);
		}
	}
}
