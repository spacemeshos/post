export CGO_ENABLED := 1
include Makefile.Inc

test: get-gpu-setup
	CGO_LDFLAGS="$(CGO_TEST_LDFLAGS)" go test -v ./...
.PHONY: test

compile-test: get-gpu-setup
	CGO_LDFLAGS="$(CGO_TEST_LDFLAGS)" go test -v -c -o $(BIN_DIR)test$(EXE) ./...
.PHONY: compile-test

ifeq ($(HOST_OS),$(filter $(HOST_OS),linux darwin))
compile-windows-test:
	CC=x86_64-w64-mingw32-gcc $(MAKE) GOOS=windows GOARCH=amd64 BIN_DIR=$(PROJ_DIR)build/ compile-test
endif
.PHONY: compile-windows-test

build: $(BIN_DIR)post$(EXE) #$(BIN_DIR)spacemesh-init$(EXE)
.PHONY: build

$(BIN_DIR)post$(EXE): get-gpu-setup
	go build -o $@ .

$(BIN_DIR)spacemesh-init$(EXE): get-gpu-setup
	cd cmd/init && go build -o $@ .

SHA = $(shell git rev-parse --short HEAD)
BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)
DOCKER_IMAGE := post:$(BRANCH)

ifeq ($(BRANCH),$(filter $(BRANCH),staging trying))
  DOCKER_IMAGE = $(DOCKER_IMAGE_REPO):$(SHA)
endif

dockerbuild-go:
	docker build -t $(DOCKER_IMAGE) .
.PHONY: dockerbuild-go
