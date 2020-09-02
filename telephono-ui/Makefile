.PHONY: all clean

all: call-buddy call-buddy-archs

clean:
	rm -f call-buddy
	rm -rf build/

call-buddy: cmd/call-buddy/*.go
	go build -o $@ $^

call-buddy-archs: cmd/call-buddy/*.go
	../build-arch.sh call-buddy $^
