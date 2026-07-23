EXCLUDED_DIRS := internal/cmd internal/version test
EXCLUSION_PATTERN = $(shell echo $(EXCLUDED_DIRS) | tr ' ' '|')
EXCLUSION_REGEX := ${PKG}/(${EXCLUSION_PATTERN})
COVERAGE_PACKAGES := $(shell ${GOCMD} list ./... | grep -v -E '$(EXCLUSION_REGEX)')

.PHONY: test
test: #T Run entire Go test suite locally, coverage report generated.
	${GOCMD} test -p 1 -coverprofile ${COVER_REPORT} -cover -run Test -tags netgo -timeout 2m -v $(COVERAGE_PACKAGES)
	${GOCMD} tool cover -func=${COVER_REPORT}

.PHONY: test-quality
test-quality: #T Go static linting (requires: https://github.com/golangci/golangci-lint).
	golangci-lint -j 2 run

.PHONY: test-vuln
test-vuln: #T Go vulnerability check (see for overview: https://go.dev/blog/vuln)
	govulncheck ./...

.PHONY: coverage
coverage: #M Analyze Go coverage profile.
	${GOCMD} tool cover -func=${COVER_REPORT}
