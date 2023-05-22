VERSION = $(file < VERSION)
export DOCKER_BUILDKIT = 1

all: release

images:
	docker build --tag cordial-build:$(VERSION) --target cordial-build .
	docker build --tag cordial --tag cordial:$(VERSION)-el8 --target cordial-run-el8 .
	docker build --tag cordial --tag cordial:$(VERSION) --target cordial-run .

release: images
	-docker rm cordial-build-$(VERSION)
	docker create --name cordial-build-$(VERSION) cordial-build:$(VERSION)
	mkdir -p releases/
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION).tar.gz releases/
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION)/bin/geneos releases/geneos-$(VERSION)
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION)/bin/geneos.exe releases/geneos-$(VERSION).exe
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION)/bin/dv2email releases/dv2email-$(VERSION)

clean:
	-docker rm cordial-build-$(VERSION)
	docker image rm cordial-build:$(VERSION)
	docker image prune --filter label=stage=cordial-build -f