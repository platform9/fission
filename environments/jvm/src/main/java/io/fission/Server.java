package io.fission;

import java.io.File;
import java.io.IOException;
import java.net.MalformedURLException;
import java.net.URL;
import java.net.URLClassLoader;
import java.util.Enumeration;
import java.util.Map;
import java.util.function.Function;
import java.util.jar.JarEntry;
import java.util.jar.JarFile;

import org.springframework.boot.*;
import org.springframework.boot.autoconfigure.*;
import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;

@RestController
@EnableAutoConfiguration
public class Server {
		
	private Function<Object, Object> fn;

	
	/*
	@GetMapping("/")
	ResponseEntity<Object> home(@RequestParam Map<String, String> params) throws JsonProcessingException {
		ObjectMapper mapper = new ObjectMapper();
		String json = "";

		json = mapper.writeValueAsString(params);
		
		if (fn == null ) {
			return ResponseEntity.badRequest().body("Container not specialized");
		} else {
			return ResponseEntity.ok(fn.apply(json));
		}
	} */
	
	@RequestMapping("/")
	ResponseEntity<Object> home(RequestEntity<?> req){
		if (fn == null ) {
			return ResponseEntity.badRequest().body("Container not specialized");
		} else {
			return ResponseEntity.ok(fn.apply(req));
		}
	}
	//TODO Add mapping for other methods too.
	

    @PostMapping(path = "/v2/specialize", consumes = "application/json")
    ResponseEntity<String> specialize(@RequestBody FunctionLoadRequest req){
        long startTime = System.nanoTime();
    	File file = new File(req.getFilepath());
    	if (!file.exists()) {
    		return ResponseEntity.badRequest().body("/userfunc/user not found");
    	}
    	
    	String entryPoint = req.getFunctionname();

    	JarFile jarFile = null;
    	ClassLoader cl = null;
    	try {
    		
    		jarFile = new JarFile(file);
    		Enumeration<JarEntry> e = jarFile.entries();
    		URL[] urls = { new URL("jar:file:" + file+"!/") };
    		//cl = URLClassLoader.newInstance(urls);
    		cl = Thread.currentThread().getContextClassLoader();

    		// Load all dependent classes from libraries etc. 
    		while (e.hasMoreElements()) {
    		    JarEntry je = e.nextElement();
    		    if(je.isDirectory() || !je.getName().endsWith(".class")){
    		        continue;
    		    }
    		    String className = je.getName().substring(0,je.getName().length()-6);
    		    className = className.replace('/', '.');
    		    cl.loadClass(className);
    		}
    		// Instantiate the function class
    		//TODO Check if it is better idea to get main class from MANIFEST.MF
			fn = (Function<Object, Object>) cl.loadClass(entryPoint).newInstance();
			
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
				//cl.close();
				jarFile.close();
			} catch (IOException e) {
				e.printStackTrace();
				return ResponseEntity.badRequest().body("Error closing the file while loading the class");
			}
		}
    	long elapsedTime = System.nanoTime() - startTime;
    	System.out.println("Specialize call done in: " + elapsedTime/1000000 +" ms");

		return ResponseEntity.ok("Done");
    }

	public static void main(String[] args) throws Exception {
		SpringApplication.run(Server.class, args);
	}

}