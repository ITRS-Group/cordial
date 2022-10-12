VERSION = $(file < VERSION)

release:
	docker build --tag cordial:$(VERSION) .
	docker create --name cordial-$(VERSION) cordial:$(VERSION)
	docker cp cordial-$(VERSION):/cordial-$(VERSION).tar.gz .
	docker cp cordial-$(VERSION):/cordial-$(VERSION).zip .
	docker rm cordial-$(VERSION)