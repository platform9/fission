def main():
	path = "/config/default/testcfgmap/TEST_KEY"
	f = open(path, "r")
	data = f.read()
	return data, 200
