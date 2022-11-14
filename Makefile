VERSION = $(file < VERSION)
export DOCKER_BUILDKIT = 1

release:
	docker build --tag cordial-build:$(VERSION) --target cordial-build .
	docker build --tag cordial --tag cordial:$(VERSION) --target cordial-run .
	docker create --name cordial-build-$(VERSION) cordial-build:$(VERSION)
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION).tar.gz .
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION).zip .
	docker rm cordial-build-$(VERSION)
	docker image rm cordial-build:$(VERSION)
	docker image prune --filter label=stage=cordial-build -f