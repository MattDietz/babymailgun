# It would be snazzy if we either had a tool that could autodiscover possible
# tasks available in the project or be able to generate them as part of the
# initial scaffolding
-include .env

# -e says exit immediately when a command fails
# -o sets pipefail, meaning if it exits with a failing command, the exit code should be of the failing command
# -u fails a bash script immediately if a variable is unset
# -x prints every command before running it
SHELL = /bin/bash -eu -o pipefail
DOCKER := docker
DOCKER_BUILD := $(DOCKER) build -t
DOCKER_TAG := $(DOCKER) tag
DOCKER_PUSH := $(DOCKER) push
DOCKER_RMI := $(DOCKER) rmi
DOCKER_COMPOSE := docker-compose -f
SUDO := sudo
DOCKERFILE := Dockerfile
PRIMARY_GROUP := $(shell id -gn)

help: # Display help
	@awk -F ':|##' \
		'/^[^\t].+?:.*?##/ {\
			printf "\033[36m%-30s\033[0m %s\n", $$1, $$NF \
		}' $(MAKEFILE_LIST)

all : build_venv build run database ## all the things
	@echo "Local dev environment created."

run :  ## docker compose everything
	if [ ! -d "./mongo_data" ]; then mkdir -p ./mongo_data; fi
	$(DOCKER_COMPOSE) docker-compose.yml up -d
	@echo "run 'make logs' to connect to docker log output"

up : ## shorthand for current environment docker-compose up -d
	$(DOCKER_COMPOSE) docker-compose.yml up -d

stop : ## teardown compose containers
	@$(DOCKER_COMPOSE) docker-compose.yml stop; \
	$(DOCKER_COMPOSE) docker-compose.yml rm -f

clean_db : stopped_database
	@$(SUDO) $(RM) -rf ./mongo_data; \

reset_web: ## teardown and recreate web container
	@$(DOCKER) stop babymailgun_mailgun_api_1; \
	$(DOCKER) rm babymailgun_mailgun_api_1; \
	$(DOCKER_COMPOSE) docker-compose.yml up -d

clean_images :
	@$(DOCKER_COMPOSE) docker-compose.yml down --rmi all

logs : 
	@$(DOCKER_COMPOSE) docker-compose.yml logs -f

.PHONY : logs clean_images reset_web clean_db stop up run all help
