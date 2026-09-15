# Vaultory — thin wrappers over docker compose, so the documented commands stay short and stable.
#
# There is no logic here beyond choosing a profile. Anything clever belongs in compose.yaml where
# it can be read.

COMPOSE := docker compose

.DEFAULT_GOAL := help
.PHONY: help up watch down logs ps reset test test-e2e shell-backend shell-db check

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[1m%-16s\033[0m %s\n", $$1, $$2}'

up: ## Start the stack (database, migrations, backend, frontend)
	$(COMPOSE) --profile dev up --build -d
	@echo
	@echo "  Vaultory is starting on http://localhost:$${VAULTORY_FRONTEND_PORT:-3000}"
	@echo
	@echo "  Authentication is out of scope for feature 001, so sign in as the seeded"
	@echo "  development collector before the collection view will show anything:"
	@echo
	@echo "    curl -c /tmp/vaultory.jar -X POST http://localhost:$${VAULTORY_FRONTEND_PORT:-3000}/api/dev/session"
	@echo
	@echo "  Follow the logs with:  make logs"

watch: ## Start the stack and sync source changes into it as you edit
	$(COMPOSE) --profile dev up --build --watch

down: ## Stop the stack. Your data is kept.
	$(COMPOSE) --profile dev down

logs: ## Follow the logs
	$(COMPOSE) --profile dev logs -f

ps: ## Show what is running
	$(COMPOSE) --profile dev ps

# Separated from `down` and named so nobody runs it by accident: this is the only command that
# destroys a developer's collectibles and uploaded images (FR-007).
reset: ## DESTROY all data — database and uploaded images — then stop
	@printf "This deletes every collectible and uploaded image. Type 'yes' to continue: " && \
		read ans && [ "$$ans" = "yes" ] || (echo "Cancelled."; exit 1)
	$(COMPOSE) --profile dev down -v

test: ## Run the backend suites (unit, integration, contract) against a throwaway database
	$(COMPOSE) --profile test run --rm --build backend-test
	@$(COMPOSE) --profile test down --remove-orphans >/dev/null 2>&1 || true

test-e2e: ## Run the browser suite. Requires `make up` first.
	$(COMPOSE) --profile e2e run --rm e2e

shell-backend: ## Open a shell in the backend container
	$(COMPOSE) --profile dev exec backend sh

shell-db: ## Open psql against the development database
	$(COMPOSE) --profile dev exec postgres \
		psql -U $${POSTGRES_USER:-vaultory} -d $${POSTGRES_DB:-vaultory_dev}

check: ## Validate the compose files without starting anything
	$(COMPOSE) --profile dev --profile test --profile e2e config --quiet
	@echo "compose.yaml is valid"
