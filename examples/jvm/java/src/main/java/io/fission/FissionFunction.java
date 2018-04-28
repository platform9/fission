package io.fission;

import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;

public interface FissionFunction<RequestEntity, ResponseEntity> {
	
	public org.springframework.http.ResponseEntity call(org.springframework.http.RequestEntity req, FissionContext context);


}
