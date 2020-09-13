.PHONY: all clean

all: launchpad

clean:
	rm -f launchpad

launchpad: cmd/launchpad/*.go
	go build -o $@ $^

install: launchpad
	cp launchpad $(prefix)/bin/
