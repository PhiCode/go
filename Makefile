.PHONY: all
all: test

.PHONY: clean
clean:
	rm -rf build

.PHONY: dev
dev:
	watch -n 5 -- make test

.PHONY: test
test:
	go test -cover $(packages)

.PHONY: bench
bench:
	go test -v -bench=.* -cpu=1,2,4,8,16 $(packages)

.PHONY: cover
cover:
	mkdir -p build
	go test -coverprofile build/coverage_pubsub.out github.com/phicode/pubsub
	go tool cover -html=build/coverage_pubsub.out -o build/coverage_pubsub.html

.PHONY: src
src:
	gofmt -w -s .
	goimports -w .
	go vet $(packages)
