.PHONY: all validate build lint format test clean

all: validate build

validate:
	./validate.sh

build:
	go build -o bin/archive ./src/cmd/archive
	go build -o bin/headlamp ./src/cmd/headlamp
	go build -o bin/index ./src/cmd/index

lint:
	golint src/...

format:
	find src/ -name "*.go" | xargs goimports -l -w

test:
	go test ./src/...

clean:
	rm -rf bin/ pkg/
