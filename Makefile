.PHONY: build install clean run container
all: build

build:
	@go build -o bin/paste main.go

run: build
	@go run main.go

install: build
	@cp bin/paste /usr/local/bin/paste

container:
	@docker build -t paste .

clean:
	@rm -f bin/paste /usr/local/bin/paste
