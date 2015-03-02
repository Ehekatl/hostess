all: build test

deps:
	@go get golang.org/x/tools/cmd/cover
	@go get golang.org/x/tools/cmd/vet
	@go get github.com/golang/lint/golint
	@go get github.com/codegangsta/cli

build: deps
	go build

test:
	go test -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	go vet
	golint

gox:
	@go get github.com/mitchellh/gox
	gox -build-toolchain

build-all: test
	@which gox || make gox
	@gox -arch="amd64" -os="darwin" -os="linux" github.com/cbednarski/hostess/cmd/hostess

install: build test
	cp hostess /usr/sbin/hostess

clean:
	rm -f ./hostess
	rm -f ./hostess_*
	rm -f ./coverage.*
