.PHONY: build
build:
	cd cmd && \
	go build -o ../glitter-boot &&\
	cd -