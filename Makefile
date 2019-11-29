.EXPORT_ALL_VARIABLES:

ifndef VERSION
VERSION := $(shell git describe --always --tags)
endif

DATE := $(shell date -u +%Y%m%d.%H%M%S)

LDFLAGS = -trimpath -ldflags "-X=main.version=$(VERSION)-$(DATE)"
CGO_ENABLED=0

targets = geottnd

.PHONY: all lint test geottnd clean

all: test $(targets)

test: CGO_ENABLED=1
test: lint
	go test -race ./...

lint:
	golangci-lint run

geottnd:
	cd cmd/geottnd && packr2 && go build $(LDFLAGS)

geottnd-image: geottnd
	cd cmd/geottnd && docker build -t geottnd:$(VERSION) .

geottnd-publish-image: geottnd-image
	docker tag  geottnd:$(VERSION) akhenakh/geottnd:latest
	docker push akhenakh/geottnd:latest

clean:
	rm -f cmd/geottnd/geottnd

generate:
	go generate ./...
