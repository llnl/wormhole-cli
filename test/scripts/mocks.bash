#!/usr/bin/env bash

# Re-build all mock files (https://github.com/uber-go/mock)

set -eo pipefail
set -o xtrace

mockgen() {
    "$(go env GOPATH)/bin/mockgen" \
        -source="internal/$1" \
        -destination="test/mocks/$2" \
        -package="$3"
}

# Registry mocks
mockgen routeregistry/service.go mock_routeregistry/mock_registry_service.go mock_routeregistry

# Requester mocks
mockgen requester/requester.go mock_requester/mock_requester.go mock_requester

# HTTP mocks
"$(go env GOPATH)/bin/mockgen" \
    -destination="test/mocks/mock_http/mock_roundtripper.go" \
    -package=mock_http \
    net/http RoundTripper
