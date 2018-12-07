NAME=swift-sftp
BINDIR=bin
GOARCH=amd64
VERSION=$(shell cat -e VERSION)

all: clean windows darwin linux

setup:
	go get -u github.com/golang/dep/cmd/dep
	dep ensure

windows:
	GOOS=$@ GOARCH=$(GOARCH) CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "-X main.version=$(VERSION)" -o $(BINDIR)/$@/$(NAME).exe

darwin:
	GOOS=$@ GOARCH=$(GOARCH) CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "-X main.version=$(VERSION)" -o $(BINDIR)/$@/$(NAME)
	cd bin/$@; gzip -c $(NAME) > $(NAME)-osx.$(GOARCH).gz

linux:
	GOOS=$@ GOARCH=$(GOARCH) CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "-X main.version=$(VERSION)" -o $(BINDIR)/$@/$(NAME)
	cd bin/$@; gzip -c $(NAME) > $(NAME)-linux.$(GOARCH).gz

rpm: linux
	rm -rf `pwd`/packaging/rpm/rpm
	mkdir -m 0777 `pwd`/packaging/rpm/rpm
	cp $(BINDIR)/linux/$(NAME) `pwd`/packaging/rpm/srv
	docker run -ti --rm -v `pwd`/packaging/rpm/srv/:/srv/ -v `pwd`/packaging/rpm/rpm:/home/builder/rpm:rw rpmbuild/centos7
	rm -f `pwd`packaging/rpm/srv/swift-sftp

deb: linux
	cd packaging/deb; dpkg-buildpackage -B -tc
#       cd packaging; dput -ol ppa:hironobu-s/swift-sftp swift-sftp_1.1_amd64.changes

clean:
	rm -rf $(BINDIR)

test:
	env ENV=test go test -cover -race -v
