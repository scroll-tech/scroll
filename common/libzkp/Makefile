.PHONY: lint docker clean libzkp

test-zkp:
	sh ./build.sh
	go test -tags ffi -timeout 0 -v

twice-test:
	sh ./build.sh
	go test -tags ffi -timeout 0 -v -times=2

test-gpu:
	sh ./build.sh
	go test -tags="gpu ffi" -timeout 0 -v

twice-gpu-test:
	sh ./build.sh
	go test -tags="gpu ffi" -timeout 0 -v -times=2