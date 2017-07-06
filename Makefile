version = $(shell git describe --tags | tr . _)
package = github.com/99designs/aws-ecr-gc

.PHONY: install
install:
	go install $(package)

.PHONY: release
release: aws-ecr-gc-$(version)-darwin-amd64.gz aws-ecr-gc-$(version)-linux-amd64.gz

%.gz: %
	gzip $<

%-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -o $@ $(package)

%-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o $@ $(package)
