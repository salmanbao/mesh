.PHONY: contracts-lint contracts-breaking contracts-generate contracts-generate-check mesh-gates services-depguard

contracts-lint:
	bash scripts/contracts-buf-lint.sh --root-path .

contracts-breaking:
	bash scripts/contracts-buf-breaking.sh --root-path .

contracts-generate:
	bash scripts/contracts-buf-generate.sh --root-path .

contracts-generate-check:
	bash scripts/contracts-buf-generate-check.sh --root-path .

mesh-gates:
	bash scripts/run-mesh-gates.sh

services-depguard:
	@set -e; \
	for dir in services/*/*; do \
		[ -d "$$dir" ] || continue; \
		echo "[depguard] $$dir"; \
		( cd "$$dir" && golangci-lint run --disable-all -E depguard ./... ); \
	done
