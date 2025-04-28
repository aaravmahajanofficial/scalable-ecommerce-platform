test-coverage:
	go test -coverprofile=coverage.out ./...
	grep -v -E "mocks/|docs/|testutils/|cmd/|internal/utils/|internal/errors/|internal/health/|internal/metrics/|pkg/stripe/|pkg/sendGrid/mocks/" coverage.out > filtered_coverage.out
	go tool cover -func=filtered_coverage.out
	go tool cover -html=filtered_coverage.out -o coverage.html