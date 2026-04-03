APP := sift
TOOLS_DIR ?= $(CURDIR)/.tmp/tools
TOOLS_BIN := $(TOOLS_DIR)/bin

.PHONY: build test race cross-build tidy vet lint lint-go lint-shell install-dev-tools smoke smoke-macos smoke-live-macos integration-live-macos smoke-windows completions install-local security-check release-dry-run package-manifests release-preflight quality-gate quality-gate-full refresh-mole-upstream

build:
	go build -o $(APP) ./cmd/sift

test:
	go test ./...

race:
	go test -race ./...

cross-build:
	mkdir -p .tmp/build
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o .tmp/build/sift-darwin-amd64 ./cmd/sift
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o .tmp/build/sift-darwin-arm64 ./cmd/sift
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o .tmp/build/sift-windows-amd64.exe ./cmd/sift
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -o .tmp/build/sift-windows-arm64.exe ./cmd/sift

vet:
	go vet ./...

install-dev-tools:
	chmod +x ./hack/install_dev_tools.sh
	./hack/install_dev_tools.sh all

lint: lint-go lint-shell

lint-go:
	chmod +x ./hack/install_dev_tools.sh
	./hack/install_dev_tools.sh staticcheck
	PATH="$(TOOLS_BIN):$$PATH" staticcheck ./...

lint-shell:
	chmod +x ./hack/install_dev_tools.sh
	./hack/install_dev_tools.sh shellcheck
	PATH="$(TOOLS_BIN):$$PATH" shellcheck -x $$(find . -path './.tmp' -prune -o -name '*.sh' -print | sort)

security-check:
	./hack/security_check.sh

smoke: smoke-macos

smoke-macos: build
	chmod +x ./hack/macos_smoke.sh
	bash -x ./hack/macos_smoke.sh ./$(APP)

smoke-live-macos: build
	chmod +x ./hack/macos_smoke.sh
	SIFT_LIVE_INTEGRATION=1 bash -x ./hack/macos_smoke.sh ./$(APP)

integration-live-macos:
	SIFT_LIVE_INTEGRATION=1 go test ./internal/platform/... ./internal/engine/... ./internal/tui/... -run LiveIntegration

smoke-windows:
	@if ! command -v pwsh >/dev/null 2>&1; then echo "pwsh is required for smoke-windows"; exit 1; fi
	mkdir -p ./bin
	GOOS=windows GOARCH=amd64 go build -o ./bin/$(APP).exe ./cmd/sift
	pwsh ./hack/windows_smoke.ps1 -Binary ./bin/$(APP).exe

quality-gate:
	./hack/security_check.sh
	$(MAKE) lint
	go test ./...
	go vet ./...
	$(MAKE) smoke
	$(MAKE) completions
	$(MAKE) cross-build
	@if command -v pwsh >/dev/null 2>&1; then $(MAKE) smoke-windows; else echo "skip: smoke-windows requires pwsh"; fi

quality-gate-full: quality-gate
	$(MAKE) race
	@if command -v goreleaser >/dev/null 2>&1; then $(MAKE) release-dry-run; else echo "skip: release-dry-run requires goreleaser"; fi

refresh-mole-upstream:
	chmod +x ./hack/refresh_mole_upstream.sh
	./hack/refresh_mole_upstream.sh

completions: build
	mkdir -p .tmp/completions
	./$(APP) completion bash > .tmp/completions/sift.bash
	./$(APP) completion zsh > .tmp/completions/_sift
	./$(APP) completion fish > .tmp/completions/sift.fish
	./$(APP) completion powershell > .tmp/completions/sift.ps1

install-local: build
	mkdir -p "$(if $(PREFIX),$(PREFIX),$$HOME/.local/bin)"
	install -m 0755 ./$(APP) "$(if $(PREFIX),$(PREFIX),$$HOME/.local/bin)/sift"
	ln -sf "$(if $(PREFIX),$(PREFIX),$$HOME/.local/bin)/sift" "$(if $(PREFIX),$(PREFIX),$$HOME/.local/bin)/si"

tidy:
	go mod tidy

release-dry-run:
	chmod +x ./hack/release_dry_run.sh ./hack/generate_package_manifests.sh ./hack/generate_homebrew_formula.sh ./hack/generate_scoop_manifest.sh ./hack/generate_winget_manifest.sh ./hack/release_preflight.sh
	./hack/release_dry_run.sh

package-manifests:
	@if [ -z "$(TAG)" ]; then echo "TAG is required, for example: make package-manifests TAG=v0.0.0-ci DIST_DIR=./.tmp/package-dist OUT_DIR=./.tmp/manifests"; exit 1; fi
	chmod +x ./hack/generate_package_manifests.sh ./hack/generate_homebrew_formula.sh ./hack/generate_scoop_manifest.sh ./hack/generate_winget_manifest.sh
	./hack/generate_package_manifests.sh "$(TAG)" "$(if $(DIST_DIR),$(DIST_DIR),./dist)" "$(if $(OUT_DIR),$(OUT_DIR),./dist/manifests)"

release-preflight:
	@if [ -z "$(TAG)" ]; then echo "TAG is required, for example: make release-preflight TAG=v0.0.0-ci DIST_DIR=./.tmp/package-dist MANIFEST_DIR=./.tmp/manifests"; exit 1; fi
	chmod +x ./hack/release_preflight.sh
	./hack/release_preflight.sh "$(TAG)" "$(if $(DIST_DIR),$(DIST_DIR),./dist)" "$(if $(MANIFEST_DIR),$(MANIFEST_DIR),./dist/manifests)"
