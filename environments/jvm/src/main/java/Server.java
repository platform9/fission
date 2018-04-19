
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
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;

@RestController
@EnableAutoConfiguration
public class Server {
		
	private Function<Object, Object> fn;

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
	}
	
	@PostMapping("/")
	ResponseEntity<Object> home(@RequestBody Object req){
		ObjectMapper mapper = new ObjectMapper();
		String json = "";
		
		try {
			json = mapper.writeValueAsString(req);
		} catch (JsonProcessingException e) {
			e.printStackTrace();
		}
		
		if (fn == null ) {
			return ResponseEntity.badRequest().body("Container not specialized");
		} else {
			return ResponseEntity.ok(fn.apply(json));
		}
	}
	

    @PostMapping(path = "/v2/specialize", consumes = "application/json")
    ResponseEntity<String> specialize(@RequestBody FunctionLoadRequest req){
        long startTime = System.nanoTime();
    	File file = new File(req.getFilepath());
    	if (!file.exists()) {
    		return ResponseEntity.badRequest().body("/userfunc/user not found");
    	}
    	System.out.println("file="+ file);
    	
    	URLClassLoader fnClass = null;
    	try {
    		
    		JarFile jarFile = new JarFile(file);
    		Enumeration<JarEntry> e = jarFile.entries();
    		URL[] urls = { new URL("jar:file:" + file+"!/") };
    		URLClassLoader cl = URLClassLoader.newInstance(urls);

    		while (e.hasMoreElements()) {
    		    JarEntry je = e.nextElement();
    		    if(je.isDirectory() || !je.getName().endsWith(".class")){
    		        continue;
    		    }
    		    String className = je.getName().substring(0,je.getName().length()-6);
    		    className = className.replace('/', '.');
    		    System.out.println("class="+className);
    		    Class c = cl.loadClass(className);

    		}

			fnClass = new URLClassLoader(new URL[] {file.toURI().toURL()});
			// How to get only the class with Function 
			fn = (Function<Object, Object>) fnClass.loadClass("io.fission.HelloWorld").newInstance();

			//fnClass.loadClass("io.fission.Person").newInstance();
			//TODO We need to load all classes
			
		} catch (MalformedURLException e) {
			e.printStackTrace();
			return ResponseEntity.badRequest().body("Error loading the class from the file");
		} catch (ClassNotFoundException e) {
			e.printStackTrace();
		} catch (InstantiationException e) {
			e.printStackTrace();
		} catch (IllegalAccessException e) {
			e.printStackTrace();
		} catch (IOException e) {
			e.printStackTrace();
		} /*finally {
			try {
				fnClass.close();
			} catch (IOException e) {
				e.printStackTrace();
				return ResponseEntity.badRequest().body("Error closing the file while loading the class");
			}
		}*/
    	long elapsedTime = System.nanoTime() - startTime;
    	System.out.println("Specialize call done in: " + elapsedTime/1000000 +" ms");

		return ResponseEntity.ok("Done");
    }

	public static void main(String[] args) throws Exception {
		SpringApplication.run(Server.class, args);
	}

}