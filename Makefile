VERSION = $(strip $(file < VERSION))
INCLUDES_VERSION = $(strip $(file < pkg/includes/VERSION))
HOLIDAYS_VERSION = $(strip $(file < pkg/holidays/VERSION))
KEYFILE = ${HOME}/.config/geneos/keyfile.aes
CREDENTIALS = ${HOME}/.config/geneos/credentials.json
NAMESPACE = docker.itrsgroup.com
SHA256SUM = sha256sum
CROSS_TARGETS = linux/amd64 linux/arm64 windows/amd64 darwin/arm64
GENERATED_DOC_DIRS = tools/geneos/docs gdna/docs tools/dv2email/docs tools/ims-gateway/docs \
	tools/gateway-reporter/docs tools/san-config/docs \
	integrations/servicenow/docs integrations/servicenow2/docs integrations/pagerduty/docs
export DOCKER_BUILDKIT = 1

.PHONY: all base build release release-fast include holidays gdna docs clean

all: base

# Production targets - all use Docker containers for sandboxing and control of environments
base:
	docker build --tag cordial --tag cordial:$(VERSION) --target cordial-run-debian .

build: 
	# Build in Docker container for environment control
	docker build --tag cordial-build:$(VERSION) --target cordial-build .

release: build gdna includes holidays docs
	$(MAKE) base
	mkdir -p release-$(VERSION)/
	-docker rm cordial-build-$(VERSION)
	docker create --name cordial-build-$(VERSION) cordial-build:$(VERSION)
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION).tar.gz release-$(VERSION)/
	docker cp cordial-build-$(VERSION):/geneos-includes-$(INCLUDES_VERSION).tar.gz release-$(VERSION)/
	docker cp cordial-build-$(VERSION):/geneos-holiday-includes-$(HOLIDAYS_VERSION).tar.gz release-$(VERSION)/
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
	cd release-$(VERSION) && $(SHA256SUM) cordial-$(VERSION).tar.gz geneos-includes-$(INCLUDES_VERSION).tar.gz geneos-holiday-includes-$(HOLIDAYS_VERSION).tar.gz > SHA256SUMS

# Build components in parallel where possible
release-fast: build gdna includes holidays docs
	@echo "All prerequisites built. Proceeding with release assembly..."
	$(MAKE) base
	mkdir -p release-$(VERSION)/
	docker build --tag cordial-build:$(VERSION) --target cordial-build .

include:
	docker build --tag geneos-includes:$(INCLUDES_VERSION) --target geneos-includes .
	mkdir -p release-$(VERSION)/
	-docker rm geneos-includes-$(INCLUDES_VERSION)
	docker create --name geneos-includes-$(INCLUDES_VERSION) geneos-includes:$(INCLUDES_VERSION)
	docker cp geneos-includes-$(INCLUDES_VERSION):/geneos-includes-$(INCLUDES_VERSION).tar.gz release-$(VERSION)/
	docker rm geneos-includes-$(INCLUDES_VERSION)

holidays:
	docker build --tag geneos-holidays-include:$(HOLIDAYS_VERSION) --target geneos-holidays-include .
	mkdir -p release-$(VERSION)/
	-docker rm geneos-holidays-include-$(HOLIDAYS_VERSION)
	docker create --name geneos-holidays-include-$(HOLIDAYS_VERSION) geneos-holidays-include:$(HOLIDAYS_VERSION)
	docker cp geneos-holidays-include-$(HOLIDAYS_VERSION):/geneos-holiday-includes-$(HOLIDAYS_VERSION).tar.gz release-$(VERSION)/
	docker rm geneos-holidays-include-$(HOLIDAYS_VERSION)

gdna:
	docker build --tag $(NAMESPACE)/$@:$(VERSION) --tag $@ --tag $@:$(VERSION) --secret id=keyfile.aes,src=${KEYFILE} --secret id=credentials.json,src=${CREDENTIALS} --target gdna . 
	docker image tag gdna $(NAMESPACE)/gdna:release

# Documentation targets
.PHONY: docs last-docs-build

last-docs-build:
	@if [ ! -f ./.last-docs-timestamp ]; then \
		for dir in $(GENERATED_DOC_DIRS); do \
			find "$$dir" -type f -name '*.md' -delete; \
		done; \
		# Build and run tools/geneos/utils/docs utility outside container (generates markdown files) \
		cd $(PWD)/tools/geneos/utils/docs && go build && ./docs; \
		# Build and run utils/docs utility outside container (generates PDFs and other artifacts) \
		cd $(PWD)/utils/docs && go build && ./docs; \
		touch ./.last-docs-timestamp; \
	else \
		TSTAMP=$$(stat -c %y ./.last-docs-timestamp | cut -d'.' -f1); \
		CURRENT=$$(date +%Y-%m-%d); \
		echo "Docs unchanged since: $$TSTAMP"; \
	fi

docs: last-docs-build

# Diagnostic and cleanup targets - all using Docker containers for environment control
.PHONY: check-version check-cross check-examples test-images clean force-rebuild build-all

check-version:
	@if ! tr -d '[:space:]' < VERSION | cmp -s - VERSION; then \
		echo "VERSION must not contain whitespace or a trailing newline" >&2; exit 1; \
	fi

# build every package for each of the supported platforms to catch
# platform specific compile errors without needing those platforms
check-cross:
	docker run --rm -v $(PWD):/app cordial-build:$(VERSION) sh -c "cd /app && go build ./..."

# the examples are a separate module with their own dependencies
check-examples:
	docker run --rm -v $(PWD):/app cordial-build:$(VERSION) sh -c "cd /app/_examples && go vet ./..."

test-images: release
	docker build --tag cordial/ubi8 --tag cordial/ubi8:$(VERSION) --target cordial-run-ubi8 .
	docker build --tag cordial/ubi9 --tag cordial/ubi9:$(VERSION) --target cordial-run-ubi9 .
	docker build --tag cordial/ubi10 --tag cordial/ubi10:$(VERSION) --target cordial-run-ubi10 .

# Force rebuild by cleaning all cached artifacts and containers
force-rebuild:
	# Clean all build artifacts and perform a full rebuild using Docker containers
	$(MAKE) clean
	docker build --tag cordial-build:$(VERSION) --target cordial-build .

build-all: check-cross
	@echo "Running cross-platform build checks..."
	$(MAKE) check-cross

clean:
	-docker rm cordial-build-$(VERSION) cordial-docs-$(VERSION) geneos-includes-$(INCLUDES_VERSION) geneos-holidays-include-$(HOLIDAYS_VERSION)
	-docker image rm cordial-build:$(VERSION) cordial-docs:$(VERSION) geneos-includes:$(INCLUDES_VERSION) geneos-holidays-include:$(HOLIDAYS_VERSION)
