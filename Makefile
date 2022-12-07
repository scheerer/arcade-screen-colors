PACKAGES_PATH = $(shell go list -f '{{ .Dir }}' ./...)

.PHONY: all
all: require tidy fmt goimports vet staticcheck test

.PHONY: require
require:
	@type "goimports" > /dev/null 2>&1 \
		|| (echo 'goimports not found: to install it, run "go install golang.org/x/tools/cmd/goimports@latest"'; exit 1)
	@type "staticcheck" > /dev/null 2>&1 \
		|| (echo 'staticcheck not found: to install it, run "go install honnef.co/go/tools/cmd/staticcheck@latest"'; exit 1)

.PHONY: tidy
tidy:
	@echo "=> Executing go mod tidy"
	@go mod tidy

.PHONY: fmt
fmt:
	@echo "=> Executing go fmt"
	@go fmt ./...

.PHONY: goimports
goimports:
	@echo "=> Executing goimports"
	@goimports -w $(PACKAGES_PATH)

.PHONY: vet
vet:
	@echo "=> Executing go vet"
	@go vet ./...

.PHONY: staticcheck
staticcheck:
	@echo "=> Executing staticcheck"
	@staticcheck ./...

.PHONY: test
test:
	@echo "=> Running tests"
	@go test ./... -covermode=atomic -coverprofile=/tmp/coverage.out -coverpkg=./... -count=1 -race -shuffle=on
	@cat .covignore | sed '/^[[:space:]]*$$/d' >/tmp/covignore
	@grep -Fvf /tmp/covignore /tmp/coverage.out  > /tmp/coverage.out-filtered
	@go tool cover -func /tmp/coverage.out-filtered | grep total: | sed -e 's/\t//g' | sed -e 's/(statements)/ /'
