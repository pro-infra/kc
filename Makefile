

build :
	go build -ldflags "-X main.version=`git describe --tags HEAD`"

all :
	echo "Compiling for every OS and Platform"
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=`git describe --tags HEAD`" -o kc.darwin_arm64
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=`git describe --tags HEAD`" -o kc.darwin_amd64
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=`git describe --tags HEAD`" -o kc.linux_amd64