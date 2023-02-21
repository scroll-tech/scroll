.PHONY: lint docker clean libzkp

ZKEVM_VERSION=3.0

zkp:
	# wget https://github.com/scroll-tech/scroll-zkevm/archive/refs/tags/${ZKEVM_VERSION}.zip

test-zkp: zkp
	go test -tags ffi -timeout 0 -v

twice-test: zkp
	go test -tags ffi -timeout 0 -v -times=2

test-gpu: zkp
	go test -tags="gpu ffi" -timeout 0 -v

twice-gpu-test: zkp
	go test -tags="gpu ffi" -timeout 0 -v -times=2