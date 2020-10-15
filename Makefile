BUILD_DATE := `date -u +%Y%m%d`
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo v0.0.1)
GIT_SHA := $(shell git rev-parse HEAD)

APP_NAME := route-server
PROJECT := github.com/gsmcwhirter/eve-route-finder

SERVER := evesite@evogames.org:~/eve-apps/
SYSTEM_DATA_FILE := ./data/systemdata.yml
SERVICE_FILE := ./route-server.service
START_SCRIPT := ./start-server.sh
INSTALLER := ./route-server-install.sh

GOPROXY ?= https://proxy.golang.org

# can specify V=1 on the line with `make` to get verbose output
V ?= 0
Q = $(if $(filter 1,$V),,@)

.DEFAULT_GOAL := help

build: version test  ## Build the binary
	$Q GOPROXY=$(GOPROXY) go build -v -ldflags "-X main.AppName=$(APP_NAME) -X main.BuildVersion=$(VERSION) -X main.BuildSHA=$(GIT_SHA) -X main.BuildDate=$(BUILD_DATE)" -o bin/$(APP_NAME) -race $(PROJECT)/cmd/$(APP_NAME)

build-release: version test  # Build the binary for linux
	$Q GOPROXY=$(GOPROXY) GOOS=linux go build -v -ldflags "-X main.AppName=$(APP_NAME) -X main.BuildVersion=$(VERSION) -X main.BuildSHA=$(GIT_SHA) -X main.BuildDate=$(BUILD_DATE)" -o bin/$(APP_NAME) $(PROJECT)/cmd/$(APP_NAME)

release: build-release
	$Q gzip -k -f bin/$(APP_NAME)
	$Q cp bin/$(APP_NAME).gz bin/$(APP_NAME)-$(VERSION).gz
	$Q tar czf bin/$(APP_NAME)-static.tar.gz eve-routes
	$Q cp bin/$(APP_NAME)-static.tar.gz bin/$(APP_NAME)-static-$(VERSION).tar.gz

clean:  ## Remove compiled artifacts
	$Q rm bin/*

deps:  ## download dependencies
	$Q GOPROXY=$(GOPROXY) go mod download

test:  ## Run the tests
	$Q GOPROXY=$(GOPROXY) go test -cover ./...

version:  ## Print the version string and git sha that would be recorded if a release was built now
	$Q echo $(VERSION) $(GIT_SHA)

upload:
	$Q scp $(SYSTEM_DATA_FILE) $(SERVICE_FILE) $(START_SCRIPT) $(INSTALLER) $(SERVER)
	$Q scp  ./bin/$(APP_NAME).gz ./bin/$(APP_NAME)-$(VERSION).gz $(SERVER)
	$Q scp $(SYSTEM_DATA_FILE) $(SERVER)systemdata.yml-$(VERSION)
	$Q scp ./bin/$(APP_NAME)-static.tar.gz ./bin/$(APP_NAME)-static-$(VERSION).tar.gz $(SERVER)

help:  ## Show the help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' ./Makefile