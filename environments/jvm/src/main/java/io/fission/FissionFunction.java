package io.fission;

import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;

public interface FissionFunction<RequestEntity, ResponseEntity> {
	
	public ResponseEntity call(RequestEntity req, FissionContext context); 

}
