GOOS := darwin
GOARCH := amd64

.PHONY: all clean deps build install

deps:
	go get -u github.com/aws/aws-sdk-go

build: deps
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build

install:
	install -m 0755 ./ana /usr/local/bin/

clean:
	rm -f ana
