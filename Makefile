.PHONY: build test agents-build agents-test platform-build platform-test

# ── agents (Go) ──────────────────────────────────────────────
agents-build:
	cd agents && go build ./...

agents-test:
	cd agents && go test ./...

# ── platform (Java/Gradle) ───────────────────────────────────
platform-common-build:
	cd platform && ./gradlew :common:build -x test

platform-common-test:
	cd platform && ./gradlew :common:test

platform-hub-build:
	cd platform && ./gradlew :hub:build -x test

platform-hub-test:
	cd platform && ./gradlew :hub:test

platform-engine-build:
	cd platform && ./gradlew :engine:build -x test

platform-engine-test:
	cd platform && ./gradlew :engine:test

# ── top-level shortcuts ──────────────────────────────────────
build: agents-build platform-common-build platform-hub-build platform-engine-build

test: agents-test platform-common-test platform-hub-test platform-engine-test

