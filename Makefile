.PHONY: ci dev

packages = github.com/PhiCode/go/...

all: install test
ci: install test bench

dev:
	watch -n 5 -- make install test

install:
	go install $(packages)

test:
	go test -cover $(packages)

bench:
	go test -bench=.* -cpu=1,2,4,8,16 $(packages)

src:
	gofmt -w -s .
	goimports -w .
