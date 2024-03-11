test:
	@go test

unit:
	@go test -tags unit

cover:
	@mkdir -p _dist
	@go test -coverprofile=_dist/coverage.out -v
	@go tool cover -html=_dist/coverage.out -o _dist/coverage.html

gen:
	@ protoc internal/*.proto --go_out=internal --go_opt=paths=source_relative \
		--go-grpc_out=internal --go-grpc_opt=paths=source_relative -I internal

cloc:
	@cloc . --exclude-dir=_example,_dist --exclude-ext=pb.go
