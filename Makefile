BUILD_DATE := `date -u +%Y%m%d`
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo v0.0.1)
GIT_SHA := $(shell git rev-parse HEAD)

APP_NAME := route-server
PROJECT := github.com/gsmcwhirter/eve-route-finder

SERVER := discordbot@evogames.org:~/eso-discord/
CONF_FILE := ./trials-bot-config.toml
SERVICE_FILE := ./eso-trials-bot.service
START_SCRIPT := ./start-bot.sh
INSTALLER := ./trials-bot-install.sh

GOPROXY ?= https://proxy.golang.org

# can specify V=1 on the line with `make` to get verbose output
V ?= 0
Q = $(if $(filter 1,$V),,@)

.DEFAULT_GOAL := help

build: version  ## Build the binary
        $Q GOPROXY=$(GOPROXY) go build -v -ldflags "-X main.AppName=$(APP_NAME) -X main.BuildVersion=$(VERSION) -X main.BuildSHA=$(GIT_SHA) -X main.BuildDate=$(BUILD_DATE)" -o bin/$(APP_NAME) -race $(PROJECT)/cmd/$(APP_NAME)

build-release-bundles: build-release
        $Q gzip -k -f bin/$(APP_NAME)
        $Q cp bin/$(APP_NAME).gz bin/$(APP_NAME)-$(VERSION).gz

clean:  ## Remove compiled artifacts
        $Q rm bin/*

release: generate test build-release-bundles  ## Release build: create a release build (disable race detection, strip symbols)

deps:  ## download dependencies
        $Q GOPROXY=$(GOPROXY) go mod download

test:  ## Run the tests
        $Q GOPROXY=$(GOPROXY) go test -cover ./...

version:  ## Print the version string and git sha that would be recorded if a release was built now
        $Q echo $(VERSION) $(GIT_SHA)

upload:
        $Q scp $(CONF_FILE) $(SERVICE_FILE) $(START_SCRIPT) $(INSTALLER) $(SERVER)
        $Q scp  ./bin/$(APP_NAME).gz ./bin/$(APP_NAME)-$(VERSION).gz $(SERVER)

help:  ## Show the help message
        @awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' ./Makefile