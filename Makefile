dependencies:
	go get -u github.com/golang/lint/golint
	go get -u github.com/sirupsen/logrus 

pkgs = ./api ./mail ./metrics ./tests ./trader ./

fmt:
	gofmt -s -l -w $(pkgs)

lint:
	golint -min_confidence=1.0 -set_exit_status $(pkgs)
