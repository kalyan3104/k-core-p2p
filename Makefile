test:
	@echo "  >  Running unit tests without race detector"
	go test ./...

test-race:
	@echo "  >  Running unit tests with race detector"
	go test -short -race ./...

