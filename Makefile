REPO=malice-plugins/get-mauled
ORG=malice
NAME=get-mauled
CATEGORY=test
VERSION=$(shell cat VERSION)

DOWNLOAD_URL?=


all: build size tag test_all

.PHONY: build
build:
	cd $(VERSION); docker build -t $(ORG)/$(NAME):$(VERSION) .

.PHONY: size
size:
	sed -i.bu 's/docker%20image-.*-blue/docker%20image-$(shell docker images --format "{{.Size}}" $(ORG)/$(NAME):$(VERSION)| cut -d' ' -f1)-blue/' README.md

.PHONY: tag
tag:
	docker tag $(ORG)/$(NAME):$(VERSION) $(ORG)/$(NAME):latest

.PHONY: tags
tags:
	docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}" $(ORG)/$(NAME)

.PHONY: ssh
ssh:
	@docker run --init -it --rm --entrypoint=sh $(ORG)/$(NAME):$(VERSION)

.PHONY: tar
tar:
	docker save $(ORG)/$(NAME):$(VERSION) -o $(NAME).tar

.PHONY: minio
minio: stop
	@echo " > Starting minio server"
	@docker run -d --name minio -p 9000:9000 -e MINIO_ACCESS_KEY=admin -e MINIO_SECRET_KEY=password minio/minio server /data
	open http://localhost:9000/minio/malice/

.PHONY: test_all
test_all: test test_minio

.PHONY: test
test:
	@echo " > ${NAME} --help"
	@docker run --rm $(ORG)/$(NAME):$(VERSION)
	@echo " > ${NAME} test"
	docker run --rm -it -v $(PWD)/tests:/malware $(ORG)/$(NAME):$(VERSION) -V download --password infected https://github.com/ytisf/theZoo/raw/master/malwares/Binaries/Duqu2/Duqu2.zip
	rm -f tests/* || true

.PHONY: test_minio
test_minio: minio
	@echo " > testing ${NAME} with minio"
	docker run --rm -it --link minio -v $(PWD)/tests:/malware $(ORG)/$(NAME):$(VERSION) -V --store-url minio:9000 --store-id admin --store-key password download --password infected https://github.com/ytisf/theZoo/raw/master/malwares/Binaries/Duqu2/Duqu2.zip
	docker run --rm -it --link minio -v $(PWD)/tests:/malware $(ORG)/$(NAME):$(VERSION) -V --store-url minio:9000 --store-id admin --store-key password malware-samples https://github.com/ytisf/theZoo/raw/master/malwares/Binaries/Duqu2/Duqu2.zip

.PHONY: stop
stop:
	@echo " > Stopping ${NAME} container"
	@docker container rm -f $(NAME) || true
	@echo " > Stopping minio container"
	@docker container rm -f minio || true

.PHONY: dry_release
dry_release:
	goreleaser --skip-publish --rm-dist --skip-validate

.PHONY: bump
bump: ## Incriment version patch number
	@echo " > Bumping VERSION"
	@hack/bump/version -p $(shell cat VERSION) > VERSION
	@git commit -am "bumping version to $(VERSION)"
	@git push

.PHONY: release
release: bump ## Create a new release from the VERSION
	@echo " > Creating Release"
	@hack/make/release $(shell cat VERSION)
	@goreleaser --rm-dist

.PHONY: circle
circle: ci-size
	@sed -i.bu 's/docker%20image-.*-blue/docker%20image-$(shell cat .circleci/size)-blue/' README.md
	@echo " > Image size is: $(shell cat .circleci/size)"

ci-build:
	@echo " > Getting CircleCI build number"
	@http https://circleci.com/api/v1.1/project/github/${REPO} | jq '.[0].build_num' > .circleci/build_num

ci-size: ci-build
	@echo " > Getting artifact sizes from CircleCI"
	@cd .circleci; rm size nsrl bloom || true
	@http https://circleci.com/api/v1.1/project/github/${REPO}/$(shell cat .circleci/build_num)/artifacts${CIRCLE_TOKEN} | jq -r ".[] | .url" | xargs wget -q -P .circleci

clean:
	docker-clean stop
	docker rmi $(ORG)/$(NAME):$(VERSION)
	docker rmi $(ORG)/$(NAME):base
	rm -rf dist

# Absolutely awesome: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := all