BINARY_NAME:=lnpayservice
OS_NAME := $(shell uname)
ifeq ($(OS_NAME), Darwin)
OPEN := open
else
OPEN := xdg-open
endif

qa: analyze test

analyze:
	@go vet ./...

test:
	@go test -cover ./...

benchmark:
	@go test -bench=. -run=^$$ ./...

coverage:
	@mkdir -p ./coverage
	@go test -coverprofile=./coverage/cover.out ./...
	@go tool cover -html=./coverage/cover.out -o ./coverage/cover.html
	@$(OPEN) ./coverage/cover.html

clean:
	@rm -rf build/

build-auto: qa clean
	@go build -o ./build/${BINARY_NAME}

build-darwin-amd64: qa clean
	@GOOS=darwin GOARCH=amd64 go build  -o ./build/${BINARY_NAME}-darwin-amd64

build-darwin-arm64: qa clean
	@GOOS=darwin GOARCH=arm64 go build -o ./build/${BINARY_NAME}-darwin-arm64

build-linux-amd64: qa clean
	@GOOS=linux GOARCH=amd64 go build -o ./build/${BINARY_NAME}-linux-amd64

build-windows-amd64: qa clean
	@GOOS=windows GOARCH=amd64 go build -o ./build/${BINARY_NAME}-amd64.exe

build: build-auto

build-all: build-darwin-amd64 build-darwin-arm64 build-linux-amd64 build-windows-amd64

.PHONY: analyze \
		benchmark \
		build \
		build-all \
		build-auto \
		build-darwin-amd64 \
		build-darwin-arm64 \
		build-linux-amd64 \
		build-windows-amd64 \
		clean \
		coverage \
		detect-version \
		qa \
		test