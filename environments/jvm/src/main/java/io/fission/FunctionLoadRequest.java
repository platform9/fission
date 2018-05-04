package io.fission;

public class FunctionLoadRequest {
	
	private String filepath;
	private String functionName;
	private String url;
	
	String getFilepath() {
		return filepath;
	}
	void setFilepath(String filepath) {
		this.filepath = filepath;
	}
	String getUrl() {
		return url;
	}
	void setUrl(String url) {
		this.url = url;
	}
	public String getFunctionName() {
		return functionName;
	}
	public void setFunctionName(String functionName) {
		this.functionName = functionName;
	}
}
