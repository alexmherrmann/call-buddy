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
	mkdir -p $(PREFIX)/bin
	cp call-buddy $(PREFIX)/bin/
	mkdir -p $(PREFIX)/lib/call-buddy/
	cp -RP build/* $(PREFIX)/lib/call-buddy

uninstall: all
	rm $(PREFIX)/bin/call-buddy
	rm -r $(PREFIX)/lib/call-buddy/
