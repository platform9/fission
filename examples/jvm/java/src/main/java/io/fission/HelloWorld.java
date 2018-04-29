package io.fission;

import java.io.IOException;

import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;

import com.fasterxml.jackson.core.JsonParseException;
import com.fasterxml.jackson.databind.JsonMappingException;
import com.fasterxml.jackson.databind.ObjectMapper;

public class HelloWorld implements FissionFunction {

	public static void main(String args[]) {
		System.out.println("Main class");
	}
	
	public HelloWorld() {
		System.out.println("Initialized the Function class");	
	}

	@Override
	public ResponseEntity call(RequestEntity req, FissionContext context) {
		
		String json = (String)req.getBody();
		
		ObjectMapper mapper = new ObjectMapper();
		Person p = null;
		try {
			p = mapper.readValue(json, Person.class);
		} catch (JsonParseException e) {
			e.printStackTrace();
		} catch (JsonMappingException e) {
			e.printStackTrace();
		} catch (IOException e) {
			e.printStackTrace();
		}
		return ResponseEntity.ok("Hello Mr. "+ p.getName() + " Happy"+ p.getAge());
		
	}

	
	
	 

}

