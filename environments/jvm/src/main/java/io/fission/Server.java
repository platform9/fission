package io.fission;

import java.io.File;
import java.io.IOException;
import java.net.MalformedURLException;
import java.net.URL;
import java.net.URLClassLoader;
import java.util.Enumeration;
import java.util.jar.JarEntry;
import java.util.jar.JarFile;
import java.util.logging.Level;
import java.util.logging.Logger;

import org.springframework.boot.*;
import org.springframework.boot.autoconfigure.*;
import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import io.fission.Function;

@RestController
@EnableAutoConfiguration
public class Server {

	private Function fn;

	private static final int CLASS_LENGTH = 6;

	private static Logger logger = Logger.getGlobal();

	@RequestMapping(value = "/", method = { RequestMethod.GET, RequestMethod.POST, RequestMethod.DELETE,
			RequestMethod.PUT })
	ResponseEntity<Object> home(RequestEntity<?> req) {
		if (fn == null) {
			return ResponseEntity.badRequest().body("Container not specialized");
		} else {
			return ((ResponseEntity<Object>) ((Function) fn).call(req, null));
		}
	}

	@PostMapping(path = "/v2/specialize", consumes = "application/json")
	ResponseEntity<String> specialize(@RequestBody FunctionLoadRequest req) {
		long startTime = System.nanoTime();
		File file = new File(req.getFilepath());
		if (!file.exists()) {
			return ResponseEntity.badRequest().body("/userfunc/user not found");
		}

		String entryPoint = req.getFunctionName();
		logger.log(Level.INFO, "Entrypoint class:" + entryPoint);
		if (entryPoint == null) {
			return ResponseEntity.badRequest().body("Entrypoint class is missing in the function");
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
				return ResponseEntity.status(500).body("Failed to initialize the class loader");
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
			return ResponseEntity.badRequest().body("Error loading the Function class file");
		} catch (ClassNotFoundException e) {
			e.printStackTrace();
			return ResponseEntity.badRequest().body("Error loading Function or dependent class");
		} catch (InstantiationException e) {
			e.printStackTrace();
			return ResponseEntity.badRequest().body("Error creating a new instance of function class");
		} catch (IllegalAccessException e) {
			e.printStackTrace();
			return ResponseEntity.badRequest().body("Error creating a new instance of function class");
		} catch (IOException e) {
			e.printStackTrace();
			return ResponseEntity.badRequest().body("Error reading the JAR file");
		} finally {
			try {
				// cl.close();
				jarFile.close();
			} catch (IOException e) {
				e.printStackTrace();
				return ResponseEntity.badRequest().body("Error closing the file while loading the class");
			}
		}
		long elapsedTime = System.nanoTime() - startTime;
		logger.log(Level.INFO, "Specialize call done in: " + elapsedTime / 1000000 + " ms");
		return ResponseEntity.ok("Done");
	}

	public static void main(String[] args) throws Exception {
		SpringApplication.run(Server.class, args);
	}
}
