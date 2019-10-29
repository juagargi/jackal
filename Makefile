install:
	@GO111MODULE=on go install -ldflags="-s -w"

install-tools:
	@GO111MODULE=on go get -u \
		golang.org/x/lint/golint \
		golang.org/x/tools/cmd/goimports

fmt: install-tools
	@echo "Checking go files format..."
	@GOIMP=$$(for f in $$(find . -type f -name "*.go" ! -path "./.cache/*" ! -path "./vendor/*" ! -name "bindata.go") ; do \
    		goimports -l $$f ; \
    	done) && echo $$GOIMP && test -z "$$GOIMP"

build:
	@echo "Building binary..."
	@GO111MODULE=on go build -ldflags="-s -w"

test:
	@echo "Running tests..."
	@GO111MODULE=on go test -race $$(go list ./...)

coverage:
	@echo "Generating coverage profile..."
	@go test -race -coverprofile=coverage.txt -covermode=atomic $$(go list ./...)

vet:
	@echo "Searching for buggy code..."
	@go vet $$(go list ./...)

lint: install-tools
	@echo "Running linter..."
	@golint $$(go list ./...)

dockerimage:
	@echo "Building binary..."
	@env GO111MODULE=on GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w"
	@echo "Building docker image..."
	@docker build -f dockerfiles/Dockerfile -t ortuman/jackal .

clean:
	@go clean
