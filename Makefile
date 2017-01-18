GOOS := darwin linux windows
GOARCH := amd64

.PHONY: all clean deps build install

deps:
	go get -u github.com/aws/aws-sdk-go

build: deps
	for i in $(GOOS); do \
		GOOS=$$i GOARCH=$(GOARCH) go build -o bin/ana_$$i; \
		cd bin/; \
		zip ana_$$i.zip ana_$$i; \
		cd -; \
	done

install:
	install -m 0755 ./bin/ana_darwin /usr/local/bin/ana

clean:
	rm -f bin/ana_*
