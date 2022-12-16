TARGETS := $(shell ls scripts)
SETUP_ENVTEST_VER := v0.0.0-20221214170741-69f093833822

.dapper:
	@echo Downloading dapper
	@curl -sL https://releases.rancher.com/dapper/latest/dapper-`uname -s`-`uname -m` > .dapper.tmp
	@@chmod +x .dapper.tmp
	@./.dapper.tmp -v
	@mv .dapper.tmp .dapper

$(TARGETS): .dapper
	./.dapper $@

.DEFAULT_GOAL := default

.PHONY: $(TARGETS)

.PHONY: install-setup-envtest
install-setup-envtest: ## Install setup-envtest.
	go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(SETUP_ENVTEST_VER)

.PHONY: setup-envtest
setup-envtest: install-setup-envtest # Build setup-envtest
	$(eval KUBEBUILDER_ASSETS := $(shell setup-envtest use --use-env -p path $(ENVTEST_K8S_VERSION)))

integration-test: setup-envtest
	KUBEBUILDER_ASSETS="$(KUBEBUILDER_ASSETS)" go test ./integrationtests/...
