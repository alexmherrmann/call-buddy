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
	mkdir -p $(prefix)/bin
	cp call-buddy $(prefix)/bin/
	mkdir -p $(prefix)/lib/call-buddy/
	cp -RP build/* $(prefix)/lib/call-buddy

uninstall: all
	rm $(prefix)/bin/call-buddy
	rm -r $(prefix)/lib/call-buddy/
