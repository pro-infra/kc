all: kc.darwin_arm64 kc.darwin_amd64 kc.linux_amd64 kc.windows_amd64

clean:
	find . -type f -a \( -name kc.darwin_arm64 -o -name kc.darwin_amd64 -o -name kc.linux_amd64 -o -name kc \) -delete	

kc.darwin_arm64: $(wildcard *.go)
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=`git describe --tags HEAD`" -o kc.darwin_arm64

kc.darwin_amd64: $(wildcard *.go)
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=`git describe --tags HEAD`" -o kc.darwin_amd64

kc.linux_amd64: $(wildcard *.go)
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=`git describe --tags HEAD`" -o kc.linux_amd64

kc.windows_amd64: $(wildcard *.go)
	GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=`git describe --tags HEAD`" -o kc.windows_amd64

.PHONY: all clean
