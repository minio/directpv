default: binary

.PHONY: binary
binary:
	@echo "Building DirectPV binary to './directpv'"
	@(cd cmd/directpv; CGO_ENABLED=0 go build --ldflags "-s -w" -o directpv)

clean:
	@echo "Cleaning up all the generated files"
	@find . -name '*.test' | xargs rm -fv
	@find . -name '*~' | xargs rm -fv
	@rm -rvf directpv

docker:
	@docker build -t minio/directpv .

swagger-gen:
	@echo "Cleaning"
	@rm -rf models
	@rm -rf restapi/operations
	@echo "Generating swagger server code from yaml"
	@swagger generate server -A directpv --main-package=management --server-package=restapi --exclude-main -P models.Principal -f ./swagger.yaml -r NOTICE