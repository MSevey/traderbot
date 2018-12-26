dependencies:
	go get -u github.com/golang/lint/golint

pkgs = ./api ./mail ./metrics ./tests ./trader ./main.go

fmt:
	gofmt -s -l -w $(pkgs)

# vet calls go vet on all packages.
vet:
	go vet $(pkgs)

lint:
	golint -min_confidence=1.0 -set_exit_status $(pkgs)