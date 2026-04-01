UNITY_ASSETS := benchmarks/unity/Assets

.PHONY: test bench-image generate bench size gen-unity

IMAGE := arpack-bench

test:
	go test ./...

bench-image:
	docker build -f Dockerfile.bench -t $(IMAGE) .

generate:
	go run ./cmd/arpack -in benchmarks/arpackmsg/messages.go -out-go benchmarks/arpackmsg
	protoc --go_out=. --go_opt=paths=source_relative benchmarks/proto/move.proto

generate-docker: bench-image
	docker run --rm -v "$(PWD):/workspace" -w /workspace $(IMAGE) make generate

bench:
	go test ./benchmarks/... -bench=. -benchmem -count=1 -run=^$$

size:
	go test ./benchmarks/... -run=TestMessageSize -v

bench-docker: bench-image
	docker run --rm -v "$(PWD):/workspace" -w /workspace $(IMAGE) make bench

gen-unity:
	mkdir -p "$(UNITY_ASSETS)/Benchmarks"
	go run ./cmd/arpack \
		-in benchmarks/arpackmsg/messages.go \
		-out-cs "$(UNITY_ASSETS)/Benchmarks/" \
		-cs-namespace Arpack.Messages
	protoc -I benchmarks/proto \
		--csharp_out="$(UNITY_ASSETS)/Benchmarks/" \
		benchmarks/proto/move.proto
	@echo "Done. Attach BenchmarkRunner to a GameObject in SampleScene and press Play."
