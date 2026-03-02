.PHONY: build test agents-build agents-test platform-build platform-test

# ── agents (Go) ──────────────────────────────────────────────
agents-build:
	cd agents && go build ./...

agents-test:
	cd agents && go test ./...

# ── platform (Java/Gradle) ───────────────────────────────────
platform-build:
	cd platform && ./gradlew build -x test

platform-test:
	cd platform && ./gradlew test

# ── top-level shortcuts ──────────────────────────────────────
build: agents-build platform-build

test: agents-test platform-test

