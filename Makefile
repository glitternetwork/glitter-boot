.PHONY: build
build:
	cd cmd/glitter-boot && \
	go build -o ../../build/glitter-boot &&\
	cd -