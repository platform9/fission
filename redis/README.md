To compile protocol buffer definitions, you need to have installed the standard C++ implementation of protocol buffers and the Go compiler plugin, protoc-gen-go.
See [here](https://github.com/golang/protobuf) for installation instructions.

Run the following command within this directory:
`protoc --go_out=build/gen *.proto`
