package io.fission;

import java.io.IOException;
import java.util.function.Function;

import com.fasterxml.jackson.core.JsonParseException;
import com.fasterxml.jackson.databind.JsonMappingException;
import com.fasterxml.jackson.databind.ObjectMapper;

public class HelloWorld implements Function<String, String> {

	public String apply(String str) {
		ObjectMapper mapper = new ObjectMapper();
		Person p = null;
		try {
			p = mapper.readValue(str, Person.class);
		} catch (JsonParseException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		} catch (JsonMappingException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		} catch (IOException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		}
		return "Hello Mr. "+ p.getName() + " Happy"+ p.getAge();
	}

}

