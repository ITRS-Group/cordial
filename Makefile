VERSION = $(strip $(file < VERSION))
KEYFILE = ${HOME}/.config/geneos/keyfile.aes
CREDENTIALS = ${HOME}/.config/geneos/credentials.json
NAMESPACE = docker.itrsgroup.com
SHA256SUM = sha256sum
CROSS_TARGETS = linux/amd64 linux/arm64 windows/amd64 darwin/arm64
GENERATED_DOC_DIRS = tools/geneos/docs gdna/docs tools/dv2email/docs tools/ims-gateway/docs \
	tools/gateway-reporter/docs tools/san-config/docs \
	integrations/servicenow/docs integrations/servicenow2/docs integrations/pagerduty/docs
export DOCKER_BUILDKIT = 1

.PHONY: all release base test-images clean build docs gdna includes check-version check-cross check-examples

all: release gdna

# VERSION is embedded in the binaries as-is, so it must be a single
# line with no whitespace, including a trailing newline
check-version:
	@if ! tr -d '[:space:]' < VERSION | cmp -s - VERSION; then \
		echo "VERSION must not contain whitespace or a trailing newline" >&2; exit 1; \
	fi

# build every package for each of the supported platforms to catch
# platform specific compile errors without needing those platforms
check-cross:
	@set -e; for target in $(CROSS_TARGETS); do \
		echo "go build ./... (GOOS=$${target%/*} GOARCH=$${target#*/})"; \
		GOOS=$${target%/*} GOARCH=$${target#*/} go build ./...; \
	done

# the examples are a separate module with their own dependencies
check-examples:
	cd _examples && go vet ./...

test-images: release
	docker build --tag cordial/ubi8 --tag cordial/ubi8:$(VERSION) --target cordial-run-ubi8 .
	docker build --tag cordial/ubi9 --tag cordial/ubi9:$(VERSION) --target cordial-run-ubi9 .
	docker build --tag cordial/ubi10 --tag cordial/ubi10:$(VERSION) --target cordial-run-ubi10 .

release: check-version docs
	$(MAKE) base
	mkdir -p release-$(VERSION)/
	-docker rm cordial-build-$(VERSION)
	docker create --name cordial-build-$(VERSION) cordial-build:$(VERSION)
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION).tar.gz release-$(VERSION)/
	docker cp cordial-build-$(VERSION):/geneos-includes-$(VERSION).tar.gz release-$(VERSION)/
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION)/bin/. release-$(VERSION)/
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION)/lib/. release-$(VERSION)/
	-docker rm cordial-docs-$(VERSION)
	docker create --name cordial-docs-$(VERSION) cordial-docs:$(VERSION)
	docker cp cordial-docs-$(VERSION):/docs/. release-$(VERSION)/docs/
	rm -f release-$(VERSION)/docs/geneos.md release-$(VERSION)/docs/geneos.pdf
	rm -f release-$(VERSION)/docs/geneos/geneos-readme.md release-$(VERSION)/docs/geneos/geneos-readme.pdf
	rm -f release-$(VERSION)/docs/geneos/geneos-documentation.* release-$(VERSION)/docs/geneos/geneos-install.* release-$(VERSION)/docs/geneos/geneos-ims.*
	rm -f release-$(VERSION)/docs/ims-gateway/SDP-AUTH.md release-$(VERSION)/docs/ims-gateway/SDP-AUTH.pdf
	rm -f release-$(VERSION)/docs/geneos/*.mermaid.md release-$(VERSION)/docs/geneos/*.mermaid-*.svg release-$(VERSION)/docs/gdna/*.mermaid.md release-$(VERSION)/docs/gdna/*.mermaid-*.svg
	for file in release-$(VERSION)/docs/geneos/*.md release-$(VERSION)/docs/geneos/*.pdf; do \
		case "$${file##*/}" in geneos-*|README.md|README.pdf) ;; *) rm -f "$$file" ;; esac; \
	done
	for file in release-$(VERSION)/docs/gdna/*.md release-$(VERSION)/docs/gdna/*.pdf; do \
		case "$${file##*/}" in gdna-[a-z]*.md|gdna-[a-z]*.pdf) ;; *) rm -f "$$file" ;; esac; \
	done
	cd release-$(VERSION) && $(SHA256SUM) cordial-$(VERSION).tar.gz geneos-includes-$(VERSION).tar.gz > SHA256SUMS

includes:
	docker build --tag geneos-includes:$(VERSION) --target geneos-includes .
	mkdir -p release-$(VERSION)/
	-docker rm geneos-includes-$(VERSION)
	docker create --name geneos-includes-$(VERSION) geneos-includes:$(VERSION)
	docker cp geneos-includes-$(VERSION):/geneos-includes-$(VERSION).tar.gz release-$(VERSION)/
	docker rm geneos-includes-$(VERSION)

build: check-version
	docker build --tag cordial-build:$(VERSION) --target cordial-build .

base: build
	docker build --tag cordial --tag cordial:$(VERSION) --target cordial-run-debian .

gdna:
	docker build --tag $(NAMESPACE)/$@:$(VERSION) --tag $@ --tag $@:$(VERSION) --secret id=keyfile.aes,src=${KEYFILE} --secret id=credentials.json,src=${CREDENTIALS} --target gdna . 
	docker image tag gdna $(NAMESPACE)/gdna:release

docs:
	@set -e; for dir in $(GENERATED_DOC_DIRS); do \
		if [ ! -f "$$dir/_DO_NOT_EDIT" ] || [ -L "$$dir/_DO_NOT_EDIT" ]; then \
			echo "Refusing to clean generated docs: $$dir/_DO_NOT_EDIT must be a regular file" >&2; \
			exit 1; \
		fi; \
	done; \
	for dir in $(GENERATED_DOC_DIRS); do \
		find "$$dir" -type f -name '*.md' -delete; \
	done
	cd tools/geneos/utils/docs && go build && ./docs
	cd utils/docs && go build && ./docs
	docker build --tag cordial-docs:$(VERSION) --target cordial-docs .

clean:
	-docker rm cordial-build-$(VERSION) cordial-docs-$(VERSION) geneos-includes-$(VERSION)
	-docker image rm cordial-build:$(VERSION) cordial-docs:$(VERSION) geneos-includes:$(VERSION)
