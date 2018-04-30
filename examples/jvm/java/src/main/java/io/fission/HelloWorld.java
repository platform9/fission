package io.fission;

import java.util.LinkedHashMap;

import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;

import com.fasterxml.jackson.databind.ObjectMapper;

import io.fission.FissionFunction;
import io.fission.FissionContext;

public class HelloWorld implements FissionFunction<RequestEntity, FissionContext> {
	
	public HelloWorld() {
		System.out.println("Initialized the Function class");	
	}

	@Override
	public ResponseEntity call(RequestEntity req, FissionContext context) {
		
		LinkedHashMap json = (LinkedHashMap) req.getBody();
		
		ObjectMapper mapper = new ObjectMapper();
		Person p = null;
		p = mapper.convertValue(json, Person.class);
		return ResponseEntity.ok("Hello Mr. "+ p.getName() + " Happy"+ p.getAge());	
	}	 

}

