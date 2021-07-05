CGO_LDFLAGS_EXT = -Wl,-rpath,$(PROJ_DIR)build
include Makefile.Inc
test: get-gpu-setup
	go test ./gpu -v

build: $(BIN_DIR)post$(EXE) #$(BIN_DIR)spacemesh-init$(EXE)

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
