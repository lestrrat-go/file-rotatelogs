.PHONY: realclean cover viewcover test lint diffs imports

realclean:
	rm coverage.out

test:
	go test -v -race ./...

cover:
	go test -v -race -coverpkg=./... -coverprofile=coverage.out.tmp ./...
	@# This is NOT cheating. tools to generate code don't need to be
	@# included in the final result
	@cat coverage.out.tmp | grep -v "internal/cmd" | grep -v "internal/codegen" > coverage.out
	@rm coverage.out.tmp

viewcover:
	go tool cover -html=coverage.out

lint:
	golangci-lint run ./...

imports:
	goimports -w ./

