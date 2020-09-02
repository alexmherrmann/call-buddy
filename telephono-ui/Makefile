.PHONY: all clean

all: call-buddy call-buddy-archs

clean:
	rm -f call-buddy
	rm -rf build/

call-buddy: cmd/call-buddy/*.go
	go build -o $@ $^

call-buddy-archs: cmd/call-buddy/*.go
	# Note that the following build names are different from the corresponding
	# GOOS-GOARCH name because we rely on the names from the following cmd:
	# "uname -s -m | tr ' ' '-' | tr '[:upper:]' '[:lower:]'"
	# which is used internally to determine the arch and os.
	#
	# Building for most servers
	env GOOS=linux GOARCH=386 go build -o build/linux-i386/call-buddy $^
	env GOOS=linux GOARCH=amd64 go build -o build/linux-x86_64/call-buddy $^
	ln -sf linux-x86_64 build/linux-amd64
	# Building for the Raspberry Pi!
	env GOOS=linux GOARCH=arm go build -o build/linux-arm/call-buddy $^
	ln -sf linux-arm build/linux-armv7l
	ln -sf linux-arm build/linux-armhf
	env GOOS=linux GOARCH=arm64 go build -o build/linux-arm64/call-buddy $^
	ln -sf linux-arm64 build/linux-armv8
	ln -sf linux-arm64 build/linux-aarch64
	# Building for MacOS
	env GOOS=darwin GOARCH=amd64 go build -o build/darwin-x86_64/call-buddy $^
	ln -sf darwin-x86_64 build/darwin-amd64
