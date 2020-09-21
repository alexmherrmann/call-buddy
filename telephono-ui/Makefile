.PHONY: all clean

all: call-buddy call-buddy-archs

clean:
	rm -f call-buddy
	rm -rf build/

call-buddy: cmd/call-buddy/*.go
	go build -o $@ $^

call-buddy-archs: cmd/call-buddy/*.go
	../build-arch.sh call-buddy $^

install: call-buddy call-buddy-archs
	mkdir -p $(realprefix)/bin
	cp call-buddy $(realprefix)/bin/
	mkdir -p $(realprefix)/lib/call-buddy/
	cp -RP build/* $(realprefix)/lib/call-buddy

uninstall: all
	rm $(realprefix)/bin/call-buddy
	rm -r $(realprefix)/lib/call-buddy/
