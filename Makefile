.PHONY: clean ci dev coverage bench cover src

packages = $(shell go list github.com/phicode/go/...)

all: install test
ci: install test bench

clean:
	rm -rf build

dev:
	watch -n 5 -- make install test

install:
	go install $(packages)

test:
	go test -cover $(packages)

bench:
	go test -v -bench=.* -cpu=1,2,4,8,16 $(packages)

cover:
	mkdir -p build
	# TODO: makefile magic to generate coverage for each package
	go test -coverprofile build/coverage_pubsub.out github.com/phicode/go/pubsub
	go tool cover -html=build/coverage_pubsub.out -o build/coverage_pubsub.html
	go test -coverprofile build/coverage_path.out github.com/phicode/go/path
	go tool cover -html=build/coverage_path.out -o build/coverage_path.html

src:
	gofmt -w -s .
	goimports -w .
	go vet $(packages)
