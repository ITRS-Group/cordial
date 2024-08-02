VERSION = $(file < VERSION)
KEYFILE = ${HOME}/.config/geneos/keyfile.aes
CREDENTIALS = ${HOME}/.config/geneos/credentials.json
NAMESPACE = docker.itrsgroup.com
export DOCKER_BUILDKIT = 1

all: release

test-images:
	docker build --tag cordial/ubi8 --tag cordial/ubi8:$(VERSION) --target cordial-run-ubi8 .
	docker build --tag cordial/ubi9 --tag cordial/ubi9:$(VERSION) --target cordial-run-ubi9 .

release: base gdna docs
	mkdir -p releases/
	-docker rm cordial-build-$(VERSION)
	docker create --name cordial-build-$(VERSION) cordial-build:$(VERSION)
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION).tar.gz releases/
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION)/bin/. releases/
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION)/docs/. releases/docs/
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION)/lib/libemail.so releases/
	docker image tag gdna $(NAMESPACE)/gdna:release

.PHONY: build gdna

build:
	docker build --tag cordial-build:$(VERSION) --target cordial-build .

base: build
	docker build --tag cordial --tag cordial:$(VERSION) --target cordial-run-debian .

gdna:
	docker build --tag $(NAMESPACE)/$@:$(VERSION) --tag $@ --tag $@:$(VERSION) --secret id=keyfile.aes,src=${KEYFILE} --secret id=credentials.json,src=${CREDENTIALS} --target gdna . 

docs:
	cd utils/docs && go build && ./docs
	cd tools/geneos/utils/docs && go build && ./docs
	cd gdna/docs && go build && ./docs

clean:
	-docker rm cordial-build-$(VERSION)
	docker image rm cordial-build:$(VERSION)
	docker image prune --filter label=stage=cordial-build -f
