VERSION = $(file < VERSION)
export DOCKER_BUILDKIT = 1

all: release

images:
	docker build --tag cordial --tag cordial:$(VERSION)-centos8 --target cordial-run-centos8 .
	docker build --tag cordial --tag cordial:$(VERSION)-centos7 --target cordial-run-centos7 .
	docker build --tag cordial --tag cordial:$(VERSION) --target cordial-run-debian .

release: images
	docker build --tag cordial-build:$(VERSION) --target cordial-build .
	-docker rm cordial-build-$(VERSION)
	docker create --name cordial-build-$(VERSION) cordial-build:$(VERSION)
	mkdir -p releases/
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION).tar.gz releases/
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION)/bin/geneos releases/geneos
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION)/bin/geneos.exe releases/geneos.exe
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION)/bin/gateway-reporter releases/gateway-reporter
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION)/bin/dv2email releases/dv2email
	docker cp cordial-build-$(VERSION):/cordial-$(VERSION)/lib/libemail.so releases/libemail.so

clean:
	-docker rm cordial-build-$(VERSION)
	docker image rm cordial-build:$(VERSION)
	docker image prune --filter label=stage=cordial-build -f