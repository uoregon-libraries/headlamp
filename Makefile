.PHONY: all validate build lint format test clean

all: vendor/src validate build

validate:
	./validate.sh

build:
	gb build

vendor/src:
	gb vendor restore

lint:
	golint src/...

format:
	find src/ -name "*.go" | xargs gofmt -l -w -s

test:
	go test ./src/...

clean:
	rm -rf bin/ pkg/
