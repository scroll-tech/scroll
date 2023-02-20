.PHONY: lint docker clean libzkp

zkp:


test-zkp: zkp
	go test -tags ffi -timeout 0 -v

twice-test: zkp
	go test -tags ffi -timeout 0 -v -times=2

test-gpu: zkp
	go test -tags="gpu ffi" -timeout 0 -v

twice-gpu-test: zkp
	go test -tags="gpu ffi" -timeout 0 -v -times=2