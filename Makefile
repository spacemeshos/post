export CGO_ENABLED := 1
include Makefile.Inc

build: postcli
.PHONY: build

test: get-gpu-setup
	@$(ULIMIT) CGO_LDFLAGS="$(CGO_TEST_LDFLAGS)" gotestsum -- -timeout 5m -p 1 -race -short ./...
.PHONY: test

compile-test: get-gpu-setup
	CGO_LDFLAGS="$(CGO_TEST_LDFLAGS)" go test -v -c -o $(BIN_DIR)test$(EXE) ./...
.PHONY: compile-test

ifeq ($(HOST_OS),$(filter $(HOST_OS),linux darwin))
compile-windows-test:
	CC=x86_64-w64-mingw32-gcc $(MAKE) GOOS=windows GOARCH=amd64 BIN_DIR=$(PROJ_DIR)build/ compile-test
endif
.PHONY: compile-windows-test

install: get-gpu-setup
	go mod download
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.50.0
	go install github.com/spacemeshos/go-scale/scalegen@v1.1.1
	go install gotest.tools/gotestsum@v1.8.2
	go install honnef.co/go/tools/cmd/staticcheck@latest
.PHONY: install

tidy:
	go mod tidy
.PHONY: tidy

test-tidy:
	# Working directory must be clean, or this test would be destructive
	git diff --quiet || (echo "\033[0;31mWorking directory not clean!\033[0m" && git --no-pager diff && exit 1)
	# We expect `go mod tidy` not to change anything, the test should fail otherwise
	make tidy
	git diff --exit-code || (git --no-pager diff && git checkout . && exit 1)
.PHONY: test-tidy

test-fmt:
	git diff --quiet || (echo "\033[0;31mWorking directory not clean!\033[0m" && git --no-pager diff && exit 1)
	# We expect `go fmt` not to change anything, the test should fail otherwise
	go fmt ./...
	git diff --exit-code || (git --no-pager diff && git checkout . && exit 1)
.PHONY: test-fmt

clear-test-cache:
	go clean -testcache
.PHONY: clear-test-cache

lint: get-gpu-setup
	go vet ./...
	./bin/golangci-lint run --config .golangci.yml
.PHONY: lint

# Auto-fixes golangci-lint issues where possible.
lint-fix: get-gpu-setup
	./bin/golangci-lint run --config .golangci.yml --fix
.PHONY: lint-fix

lint-github-action: get-gpu-setup
	go vet ./...
	./bin/golangci-lint run --config .golangci.yml --out-format=github-actions
.PHONY: lint-github-action

cover: get-gpu-setup
	@$(ULIMIT) CGO_LDFLAGS="$(CGO_TEST_LDFLAGS)" go test -coverprofile=cover.out -timeout 0 -p 1 ./...
.PHONY: cover

staticcheck: get-gpu-setup
	@$(ULIMIT) CGO_LDFLAGS="$(CGO_TEST_LDFLAGS)" staticcheck ./...
.PHONY: staticcheck

generate: get-gpu-setup
	@$(ULIMIT) CGO_LDFLAGS="$(CGO_TEST_LDFLAGS)" go generate ./...
.PHONY: generate

postcli: get-gpu-setup
	go build -o $(BIN_DIR)$@$(EXE) ./cmd/postcli
.PHONY: postcli
