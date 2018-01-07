DOCKER := docker
DOCKER_COMPOSE := docker-compose -f
SUDO := sudo

help: # Display help
	@awk -F ':|##' \
		'/^[^\t].+?:.*?##/ {\
			printf "\033[36m%-30s\033[0m %s\n", $$1, $$NF \
		}' $(MAKEFILE_LIST)

run :  ## docker compose everything
	$(DOCKER_COMPOSE) docker-compose.yml up -d
	@echo "run 'make logs' to connect to docker log output"

install_python : ## Sets up python deps in a venv so you can use the CLI
	@pip install -r python_src/requirements.txt
	@pip install -e python_src
	@cd python_src/ && tox

stop : ## teardown compose containers
	@$(DOCKER_COMPOSE) docker-compose.yml stop; \
	$(DOCKER_COMPOSE) docker-compose.yml rm -f

reset_web: ## teardown and recreate web container
	@$(DOCKER) stop babymailgun_mailgun_api_1; \
	$(DOCKER) rm babymailgun_mailgun_api_1; \
	$(DOCKER_COMPOSE) docker-compose.yml up -d

logs : ## Helper for connecting to the docker-compose log output
	@$(DOCKER_COMPOSE) docker-compose.yml logs -f

python_tests : ## Helper for running the python tests
	@cd python_src && tox

golang_tests : ## Helper for kicking off the golang tests
	@cd golang_src && go test -cover -v ./ ./cmd
 
functional_tests : ## Helper for running the functional test suite
	@cd python_src && tox -e functional

shell : ## Runs the API container as an interactive shell for access to the CLI
	@echo "Simply run 'mailgun_cli' to see the list of available commands, or 'tox' to run tests"
	@docker-compose exec mailgun_api sh

.PHONY : logs install_python reset_web clean_db stop run help python_tests functional_tests shell
