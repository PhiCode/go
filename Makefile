.PHONY: clean ci dev coverage bench cover src

all: install test
ci: install test bench

clean:
	rm -rf build

dev:
	watch -n 5 -- make test

test:
	go test -cover $(packages)

bench:
	go test -v -bench=.* -cpu=1,2,4,8,16 $(packages)

cover:
	mkdir -p build
	go test -coverprofile build/coverage_pubsub.out github.com/phicode/pubsub
	go tool cover -html=build/coverage_pubsub.out -o build/coverage_pubsub.html

src:
	gofmt -w -s .
	goimports -w .
	go vet $(packages)
