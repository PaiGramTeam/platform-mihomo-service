GOCMD := go
PROTOC := protoc

.PHONY: proto test fmt vet run

proto:
	$(PROTOC) --go_out=. --go_opt=paths=source_relative internal/conf/conf.proto
	$(if $(wildcard api/mihomo/v1/mihomo_account.proto),$(PROTOC) --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api/mihomo/v1/mihomo_account.proto,)

test:
	$(GOCMD) test ./...

fmt:
	$(GOCMD) fmt ./...

vet:
	$(GOCMD) vet ./...

run:
	$(GOCMD) run ./cmd/platform-mihomo-service --conf configs/config.yaml
