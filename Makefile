NAME=swift-sftp
BINDIR=bin
GOARCH=amd64

all: clean windows darwin linux

setup:
	go get -u github.com/golang/dep/cmd/dep
	dep ensure

windows:
	GOOS=$@ GOARCH=$(GOARCH) CGO_ENABLED=0 go build $(GOFLAGS) -o $(BINDIR)/$@/$(NAME).exe

darwin:
	GOOS=$@ GOARCH=$(GOARCH) CGO_ENABLED=0 go build $(GOFLAGS) -o $(BINDIR)/$@/$(NAME)
	cd bin/$@; gzip -c $(NAME) > $(NAME)-osx.$(GOARCH).gz

linux:
	GOOS=$@ GOARCH=$(GOARCH) CGO_ENABLED=0 go build $(GOFLAGS) -o $(BINDIR)/$@/$(NAME)
	cd bin/$@; gzip -c $(NAME) > $(NAME)-linux.$(GOARCH).gz

clean:
	rm -rf $(BINDIR)

test:
	env ENV=test go test -cover -race -v
