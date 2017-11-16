def main():
	path = "/secrets/default/testsecret/TEST_KEY"
	f = open(path, "r")
	data = f.read()
	#print()
	return data, 200
