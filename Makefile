.PHONY: all app windows clean test

all:
	./scripts/build.sh all

app:
	./scripts/build.sh app

windows:
	./scripts/build.sh windows

test:
	go test -count=1 -race -v ./...

clean:
	./scripts/build.sh clean
