.PHONY: ci dev coverage bench src

packages = github.com/phicode/go/...

all: install test
ci: install test bench

dev:
	watch -n 5 -- make install test

install:
	go install $(packages)

test:
	go test -cover $(packages)

bench:
	go test -v -bench=.* -cpu=1,2,4,8,16 $(packages)

#TODO: create and include generic go makefile for these kind of tasks
#coverage:
#	mkdir -p coverage
#	go get -u -v github.com/axw/gocov/gocov
#	go get -u -v github.com/AlekSi/gocov-xml
#	go test -coverprofile=coverage/pubsub.out github.com/phicode/go/pubsub
#	gocov convert coverage/pubsub.out > coverage/pubsub.json
#	gocov-xml < coverage/pubsub.json > coverage/pubsub.xml

src:
	gofmt -w -s .
	goimports -w .
