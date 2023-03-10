PROJECTNAME := zongzi

test:
	@go test ./...

cover:
	@mkdir -p _dist
	@go test ./... -coverprofile=_dist/coverage.out
	@go tool cover -html=_dist/coverage.out -o _dist/coverage.html

gen:
	@ protoc pb/*.proto --go_out=pb --go_opt=paths=source_relative -I pb