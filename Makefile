export CGO_ENABLED := 1
include Makefile.Inc

GOLANGCI_LINT_VERSION := v1.57.0
STATICCHECK_VERSION := v0.4.7
GOTESTSUM_VERSION := v1.11.0
GOSCALE_VERSION := v1.1.13
MOCKGEN_VERSION := v0.4.0

build: postcli
.PHONY: build

test: get-postrs-lib
	@$(ULIMIT) CGO_LDFLAGS="$(CGO_TEST_LDFLAGS)" gotestsum -- -timeout 5m -p 1 -race -short ./...
.PHONY: test

compile-test: get-postrs-lib
	CGO_LDFLAGS="$(CGO_TEST_LDFLAGS)" go test -v -c -o $(BIN_DIR)test$(EXE) ./...
.PHONY: compile-test

ifeq ($(HOST_OS),$(filter $(HOST_OS),linux darwin))
compile-windows-test:
	CC=x86_64-w64-mingw32-gcc $(MAKE) GOOS=windows GOARCH=amd64 BIN_DIR=$(PROJ_DIR)build/ compile-test
endif
.PHONY: compile-windows-test

install: get-postrs-lib
	go mod download
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s $(GOLANGCI_LINT_VERSION)
	go install gotest.tools/gotestsum@$(GOTESTSUM_VERSION)
	go install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
	go install go.uber.org/mock/mockgen@$(MOCKGEN_VERSION)
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

lint: get-postrs-lib
	./bin/golangci-lint run --config .golangci.yml
.PHONY: lint

# Auto-fixes golangci-lint issues where possible.
lint-fix: get-postrs-lib
	./bin/golangci-lint run --config .golangci.yml --fix
.PHONY: lint-fix

lint-github-action: get-postrs-lib
	./bin/golangci-lint run --config .golangci.yml --out-format=github-actions
.PHONY: lint-github-action

cover: get-postrs-lib
	@$(ULIMIT) CGO_LDFLAGS="$(CGO_TEST_LDFLAGS)" go test -coverprofile=cover.out -timeout 0 -p 1 -coverpkg=./... ./...
.PHONY: cover

staticcheck: get-postrs-lib
	@$(ULIMIT) CGO_LDFLAGS="$(CGO_TEST_LDFLAGS)" staticcheck ./...
.PHONY: staticcheck

generate: get-postrs-lib
	@$(ULIMIT) CGO_LDFLAGS="$(CGO_TEST_LDFLAGS)" go generate ./...
.PHONY: generate

test-generate:
	@git diff --quiet || (echo "\033[0;31mWorking directory not clean!\033[0m" && git --no-pager diff && exit 1)
	@make generate
	@git diff --name-only --diff-filter=AM --exit-code . || { echo "\nPlease rerun 'make generate' and commit changes.\n"; exit 1; }
.PHONY: test-generate

postcli: get-postrs-lib
	go build -o $(BIN_DIR)$@$(EXE) ./cmd/postcli
.PHONY: postcli

bench:
	@$(ULIMIT) CGO_LDFLAGS="$(CGO_TEST_LDFLAGS)" go test -benchmem -run='^$$' -bench 'BenchmarkVerifying|BenchmarkProving' github.com/spacemeshos/post/proving github.com/spacemeshos/post/verifying
.PHONY: bench
