
all: README.md
	go build ./...

README.md: go-types.md weave.go $(wildcard */*.go)
	go run weave.go go-types.md >README.md

# This is for previewing using github.
# $HOME/markdown must be a github client.
test: README.md
	cp README.md $$HOME/markdown/
	(cd $$HOME/markdown/ && git commit -m . README.md && git push)

