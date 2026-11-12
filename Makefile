# Vaultory — thin wrappers over docker compose, so the documented commands stay short and stable.
#
# There is no logic here beyond choosing a profile. Anything clever belongs in compose.yaml where
# it can be read.

COMPOSE := docker compose

.DEFAULT_GOAL := help
.PHONY: help up watch down logs ps reset test test-e2e shell-backend shell-db check migrate-to \
	one-stack-per-checkout

# compose.yaml pins the project name, and Compose resolves containers, networks and volumes by
# that name rather than by directory — so every checkout on this machine drives the same stack. In
# a linked worktree that is almost never what was meant: `make up` there does not start a stack for
# the worktree, it adopts this one's and migrates its database to whatever the branch carries. That
# happened on 29 October and left the main checkout unable to start (#87).
#
# A worktree is exactly detectable: its git dir is not the common one. Asked of git first, and of
# the filesystem when git has nothing to say — a command that is missing or fails answers with two
# empty strings, which compare equal, and a guard that reads that as "not a worktree" would wave
# `make reset` through against the main checkout's data. What makes a linked worktree one is that
# its `.git` is a regular file naming the real git dir, and that is true whether or not git can be
# run at all.
#
# Refused rather than named automatically, because a different default would move everyone's
# existing volumes, and because the second stack wants the same host ports — which is the
# developer's decision, not ours.
one-stack-per-checkout:
	@if [ -n "$$COMPOSE_PROJECT_NAME" ]; then exit 0; fi; \
	gitdir=$$(git rev-parse --git-dir 2>/dev/null); \
	common=$$(git rev-parse --git-common-dir 2>/dev/null); \
	if [ -n "$$gitdir" ] && [ -n "$$common" ]; then \
		[ "$$gitdir" != "$$common" ] || exit 0; \
	else \
		[ -f "$(CURDIR)/.git" ] || exit 0; \
	fi; \
	{ \
		echo "This is a linked worktree, and compose.yaml pins one project name, so this would"; \
		echo "drive the main checkout's containers and its development database — not a stack"; \
		echo "of its own. Give this worktree its own:"; \
		echo; \
		echo "  COMPOSE_PROJECT_NAME=vaultory-$$(basename "$$PWD") make $(or $(MAKECMDGOALS),up)"; \
		echo; \
		echo "The ports are the same either way, so stop the other stack first, or set"; \
		echo "VAULTORY_*_PORT in a .env beside compose.yaml."; \
		exit 2; \
	}

help: ## Show this help
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[1m%-16s\033[0m %s\n", $$1, $$2}'

up: one-stack-per-checkout ## Start the stack (database, migrations, backend, frontend)
	@# When the migrations step fails, its own explanation is the useful part — and `up -d` does not
	@# print it. Show that step's output, and only when that step is what failed, so an unrelated
	@# failure is not buried under it (feature 012).
	@#
	@# "That step" means this invocation's: the container can be left over from an earlier run, with
	@# its exit code and log still readable (#70). Docker's own StartedAt says whether it ran now.
	@# It changes on every start and carries nanoseconds, where `date` here gives whole seconds and
	@# two runs inside the same second are indistinguishable by it (#81). The same value bounds the
	@# log, because starting a container again keeps the earlier run's lines in front of the new
	@# ones. An absent container inspects to nothing, which differs from anything and so reads as
	@# "ran now" — correct, since there was nothing to be stale.
	@before=$$(docker inspect -f '{{.State.StartedAt}}' \
		"$$($(COMPOSE) --profile dev ps -aq migrate 2>/dev/null | head -n1)" 2>/dev/null); \
	$(COMPOSE) --profile dev up --build -d || { \
		status=$$?; \
		id=$$($(COMPOSE) --profile dev ps -aq migrate 2>/dev/null | head -n1); \
		started=$$(docker inspect -f '{{.State.StartedAt}}' "$$id" 2>/dev/null); \
		if [ -n "$$started" ] && [ "$$started" != "$$before" ] && \
			[ "$$(docker inspect -f '{{.State.ExitCode}}' "$$id" 2>/dev/null)" != "0" ]; then \
			$(COMPOSE) --profile dev logs --since "$$started" --no-log-prefix migrate; \
		fi; \
		exit $$status; \
	}
	@echo
	@echo "  Vaultory is starting on http://localhost:$${VAULTORY_FRONTEND_PORT:-3000}"
	@echo
	@echo "  Create an account to get a vault. There is no development sign-in any more:"
	@echo "  feature 004 replaced it, and /api/dev/session now returns 404."
	@echo
	@echo "  Follow the logs with:  make logs"

watch: one-stack-per-checkout ## Start the stack and sync source changes into it as you edit
	$(COMPOSE) --profile dev up --build --watch

down: one-stack-per-checkout ## Stop the stack. Your data is kept.
	$(COMPOSE) --profile dev down

logs: ## Follow the logs
	$(COMPOSE) --profile dev logs -f

ps: ## Show what is running
	$(COMPOSE) --profile dev ps

# Separated from `down` and named so nobody runs it by accident: this is the only command that
# destroys a developer's collectibles and uploaded images (FR-007).
reset: one-stack-per-checkout ## DESTROY all data — database and uploaded images — then stop
	@printf "This deletes every collectible and uploaded image. Type 'yes' to continue: " && \
		read ans && [ "$$ans" = "yes" ] || (echo "Cancelled."; exit 1)
	$(COMPOSE) --profile dev down -v

test: one-stack-per-checkout ## Run the backend suites (unit, integration, contract) against a throwaway database
	$(COMPOSE) --profile test run --rm --build backend-test
	@$(COMPOSE) --profile test down --remove-orphans >/dev/null 2>&1 || true

test-e2e: one-stack-per-checkout ## Run the browser suite. Requires `make up` first.
	@# Both profiles: the e2e container shares the frontend's network namespace, so that service
	@# has to be part of the project Compose is resolving. It already had to be running anyway.
	$(COMPOSE) --profile dev --profile e2e run --rm e2e

# For a database ahead of the checkout (feature 012): run from the branch whose migrations are
# applied, since only it has the files that undo them. Never runs without a version, and never on
# its own — `make up` only ever suggests it.
migrate-to: one-stack-per-checkout ## Move the dev database to VERSION using this checkout's migrations, e.g. VERSION=10
	@[ -n "$(VERSION)" ] || { echo "Say which version: make migrate-to VERSION=10"; exit 2; }
	$(COMPOSE) --profile dev run --rm migrate goto $(VERSION)

shell-backend: ## Open a shell in the backend container
	$(COMPOSE) --profile dev exec backend sh

shell-db: ## Open psql against the development database
	$(COMPOSE) --profile dev exec postgres \
		psql -U $${POSTGRES_USER:-vaultory} -d $${POSTGRES_DB:-vaultory_dev}

prod-build: ## Build the production images. Verification only — this is not a deployment.
	@# Dummy values on purpose. compose.prod.yaml marks these required with ${VAR:?}, and Compose
	@# interpolates the whole file even for `build`, so a build fails without them. Building needs
	@# no secrets; only `up` does, and there the :? guard still bites.
	POSTGRES_PASSWORD=build-only \
	VAULTORY_SESSION_SECRET=build-only-secret-at-least-32-chars \
	VAULTORY_AUTH_DB_PASSWORD=build-only \
	VAULTORY_PUBLIC_URL=http://build-only.invalid \
	MAIL_HOST=build-only.invalid MAIL_FROM=build-only@invalid \
	MAIL_USER=build-only MAIL_PASSWORD=build-only \
		docker compose -f compose.prod.yaml build
	@echo
	@docker image ls --format '{{.Repository}}:{{.Tag}}\t{{.Size}}' | grep '^vaultory-prod' || true

check: ## Validate the compose files without starting anything
	$(COMPOSE) --profile dev --profile test --profile e2e config --quiet
	@echo "compose.yaml is valid"
	@POSTGRES_PASSWORD=check \
	VAULTORY_SESSION_SECRET=check-only-secret-at-least-32-characters \
	VAULTORY_AUTH_DB_PASSWORD=check \
	VAULTORY_PUBLIC_URL=http://check.invalid \
	MAIL_HOST=check.invalid MAIL_FROM=check@invalid \
	MAIL_USER=check MAIL_PASSWORD=check \
		docker compose -f compose.prod.yaml config --quiet
	@echo "compose.prod.yaml is valid"
