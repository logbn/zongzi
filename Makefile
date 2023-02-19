PROJECTNAME := zongzi

test:
	@go test ./...

cover:
	@mkdir -p _dist
	@go test ./... -coverprofile=_dist/coverage.out
	@go tool cover -html=_dist/coverage.out -o _dist/coverage.html
