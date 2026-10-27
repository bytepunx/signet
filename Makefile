.PHONY: check test test-server test-proto test-client-go test-client-python test-client-typescript test-client-rust test-client-csharp test-client-elixir integration smoke board stats dashboard dashboard-stop lint-md

check: ## Validate the repo against the system-flow standard
	scripts/check.sh

test: ## Behavior tests, fast, run on every iteration
	scripts/test.sh

test-server: ## Behavior tests, lint, build, and chart checks for the server
	scripts/test-server.sh

test-proto: ## buf lint and buf breaking for the proto module
	scripts/test-proto.sh

test-client-go: ## Behavior tests for the Go client
	scripts/test-client-go.sh

test-client-python: ## Behavior tests for the Python client
	scripts/test-client-python.sh

test-client-typescript: ## Behavior tests for the TypeScript client
	scripts/test-client-typescript.sh

test-client-rust: ## Behavior tests for the Rust client
	scripts/test-client-rust.sh

test-client-csharp: ## Behavior tests for the C# client
	scripts/test-client-csharp.sh

test-client-elixir: ## Behavior tests for the Elixir client
	scripts/test-client-elixir.sh

integration: ## Integration tests, after test passes
	scripts/integration.sh

smoke: ## Smoke tests, after integration passes
	scripts/smoke.sh

lint-md: ## Lint markdown
	scripts/lint-md.sh

board: ## Print the kanban board
	flai board

stats: ## Print flow metrics
	flai stats

dashboard: ## Run the flaiover dashboard against this repo
	flai dashboard

dashboard-stop:
	flai dashboard stop

help:
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-16s %s\n", $$1, $$2}'
