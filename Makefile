SHELL := /bin/sh

GO ?= go
CGO_ENABLED ?= 1
GOFLAGS ?= -buildvcs=false
export GOFLAGS

APP_NAME ?= dutyround
BIN_DIR ?= bin
SERVER_BIN ?= $(BIN_DIR)/dutyround
SEED_BIN ?= $(BIN_DIR)/seed

PORT ?= 8080
DB_PATH ?= dutyround.db
SESSION_SECRET ?= dev-secret-change-in-production
GIN_MODE ?= debug

SEED_NAME ?= Admin
SEED_EMAIL ?= admin@example.com
SEED_PASSWORD ?= password123

IMAGE ?= $(APP_NAME)
TAG ?= latest

PLAYWRIGHT_BIN ?= ./node_modules/.bin/playwright
PLAYWRIGHT ?= $(PLAYWRIGHT_BIN) test
E2E_SESSION_SECRET ?= e2e-secret
E2E_SEED_NAME ?= Admin
E2E_SEED_PASSWORD ?= password123
E2E_DB_DIR ?= /tmp
E2E_LOG_DIR ?= /tmp
E2E_CONFIG_DIR ?= /tmp

.PHONY: help
help:
	@printf '%s\n' 'Available targets:'
	@printf '  %-26s %s\n' 'make test' 'Run the Go test suite'
	@printf '  %-26s %s\n' 'make test-all' 'Run Go tests and all Playwright e2e specs'
	@printf '  %-26s %s\n' 'make test-unit' 'Run all Go tests'
	@printf '  %-26s %s\n' 'make test-domain' 'Run domain package tests'
	@printf '  %-26s %s\n' 'make test-handler' 'Run handler package tests'
	@printf '  %-26s %s\n' 'make test-service' 'Run service package tests'
	@printf '  %-26s %s\n' 'make test-repository' 'Run sqlite repository tests'
	@printf '  %-26s %s\n' 'make test-race' 'Run Go tests with the race detector'
	@printf '  %-26s %s\n' 'make test-cover' 'Run Go tests with coverage output'
	@printf '  %-26s %s\n' 'make test-e2e' 'Run all Playwright e2e specs'
	@printf '  %-26s %s\n' 'make e2e-profile' 'Run profile occurrence e2e spec'
	@printf '  %-26s %s\n' 'make e2e-security' 'Run profile security e2e spec'
	@printf '  %-26s %s\n' 'make e2e-bugs' 'Run profile bug-regression e2e spec'
	@printf '  %-26s %s\n' 'make e2e-install' 'Install Playwright test package and browsers'
	@printf '  %-26s %s\n' 'make run' 'Run the web server locally'
	@printf '  %-26s %s\n' 'make seed' 'Seed an initial admin account'
	@printf '  %-26s %s\n' 'make build' 'Build server and seed binaries'
	@printf '  %-26s %s\n' 'make build-image' 'Build the Docker image'
	@printf '  %-26s %s\n' 'make clean' 'Remove local build artifacts and coverage'

.PHONY: test test-all check
test: test-unit

test-all: test-unit test-e2e

check: vet test-unit

.PHONY: test-unit test-go test-domain test-handler test-service test-repository test-race test-cover vet
test-unit test-go:
	$(GO) test ./...

test-domain:
	$(GO) test ./internal/domain

test-handler:
	$(GO) test ./internal/handler

test-service:
	$(GO) test ./internal/service

test-repository:
	$(GO) test ./internal/repository/sqlite

test-race:
	$(GO) test -race ./...

test-cover:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

vet:
	$(GO) vet ./...

.PHONY: build
build: $(BIN_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o $(SERVER_BIN) ./cmd/server
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o $(SEED_BIN) ./cmd/seed

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

.PHONY: run seed
run:
	DB_PATH="$(DB_PATH)" PORT="$(PORT)" SESSION_SECRET="$(SESSION_SECRET)" GIN_MODE="$(GIN_MODE)" $(GO) run ./cmd/server

seed:
	DB_PATH="$(DB_PATH)" $(GO) run ./cmd/seed -name="$(SEED_NAME)" -email="$(SEED_EMAIL)" -password="$(SEED_PASSWORD)"

.PHONY: image build-image docker-build
image build-image docker-build:
	docker build -t $(IMAGE):$(TAG) .

.PHONY: e2e-install test-e2e e2e e2e-profile e2e-security e2e-bugs
e2e-install:
	npm install --no-save --no-package-lock @playwright/test
	$(PLAYWRIGHT_BIN) install

test-e2e e2e: e2e-profile e2e-security e2e-bugs

e2e-profile: $(PLAYWRIGHT_BIN)
	$(call run_e2e,e2e-profile,3991,admin@test.com,profile-occurrences.spec.ts)

e2e-security: $(PLAYWRIGHT_BIN)
	$(call run_e2e,e2e-security,3992,secadmin@test.com,profile-security.spec.ts)

e2e-bugs: $(PLAYWRIGHT_BIN)
	$(call run_e2e,e2e-bugs,3993,bugadmin@test.com,profile-bugs.spec.ts)

$(PLAYWRIGHT_BIN):
	npm install --no-save --no-package-lock @playwright/test

define run_e2e
	@set -eu; \
	name="$(1)"; \
	port="$(2)"; \
	email="$(3)"; \
	spec="$(4)"; \
	test_dir="$$(pwd)/e2e"; \
	db="$(E2E_DB_DIR)/dutyround-$${name}.db"; \
	log="$(E2E_LOG_DIR)/dutyround-$${name}.log"; \
	config="$(E2E_CONFIG_DIR)/dutyround-$${name}.playwright.config.cjs"; \
	base_url="http://localhost:$${port}"; \
	printf 'Preparing %s on %s\n' "$${name}" "$${base_url}"; \
	rm -f "$${db}" "$${db}-shm" "$${db}-wal"; \
	printf '%s\n' \
		'module.exports = {' \
		"  testDir: '$${test_dir}'," \
		'  use: {' \
		"    baseURL: '$${base_url}'," \
		'  },' \
		'};' > "$${config}"; \
	DB_PATH="$${db}" PORT="$${port}" SESSION_SECRET="$(E2E_SESSION_SECRET)" GIN_MODE=debug $(GO) run ./cmd/server >"$${log}" 2>&1 & \
	pid="$$!"; \
	cleanup() { kill "$${pid}" >/dev/null 2>&1 || true; wait "$${pid}" >/dev/null 2>&1 || true; rm -f "$${config}"; }; \
	trap cleanup EXIT INT TERM; \
	for _ in $$(seq 1 120); do \
		if ! kill -0 "$${pid}" >/dev/null 2>&1; then \
			cat "$${log}"; \
			exit 1; \
		fi; \
		if curl -fsS "$${base_url}/login" >/dev/null 2>&1; then \
			break; \
		fi; \
		sleep 0.5; \
	done; \
	if ! curl -fsS "$${base_url}/login" >/dev/null 2>&1; then \
		cat "$${log}"; \
		exit 1; \
	fi; \
	DB_PATH="$${db}" $(GO) run ./cmd/seed -name="$(E2E_SEED_NAME)" -email="$${email}" -password="$(E2E_SEED_PASSWORD)" >/dev/null; \
	$(PLAYWRIGHT) "$${spec}" --config="$${config}"
endef

.PHONY: clean
clean:
	rm -rf $(BIN_DIR) coverage.out
