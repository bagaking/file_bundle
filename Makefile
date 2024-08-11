.PHONY: check fmt fmt-check test

GOFILES := $(shell find . -name '*.go' -not -path './vendor/*')

check: fmt-check test

fmt:
	gofmt -w $(GOFILES)

fmt-check:
	@unformatted="$$(gofmt -l $(GOFILES))"; \
	if [ -n "$$unformatted" ]; then \
		echo "Go files need formatting:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

test:
	go test ./...
