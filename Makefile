run = .
pkgs = ./api ./mail ./metrics ./tests ./trader ./

dependencies:
	# General dependencies
	go get -u github.com/golang/lint/golint
	go get -u github.com/sirupsen/logrus 
	go get -u github.com/NebulousLabs/glyphcheck

dev:
	go install $(pkgs)

fmt:
	gofmt -s -l -w $(pkgs)

lint:
	golint -min_confidence=1.0 -set_exit_status $(pkgs)

test:
	go test -short -timeout=5s $(pkgs) -run=$(run)

# vet calls go vet on all packages.
# NOTE: go vet requires packages to be built in order to obtain type info.
vet:
	go vet $(pkgs)