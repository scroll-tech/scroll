.PHONY: lint docker clean libzkp

test-zkp:
	bash build.sh
	go test -tags ffi -timeout 0 -v

twice-test:
	bash build.sh
	go test -tags ffi -timeout 0 -v -times=2

test-gpu:
	bash build.sh
	go test -tags="gpu ffi" -timeout 0 -v

twice-gpu-test:
	bash build.sh
	go test -tags="gpu ffi" -timeout 0 -v -times=2