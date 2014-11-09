ASSETS=config.json.sample mess.sql static/ static/bootstrap/ static/bootstrap/mixins/ static/fonts/bootstrap/ static/javascripts/bootstrap/ template/

env:
	mkdir -p env/bin env/pkg env/src/github.com/natmeox
	ln -s ../../../.. env/src/github.com/natmeox/mess
	GOPATH=`pwd`/env go get github.com/natmeox/mess
	GOPATH=`pwd`/env go get github.com/jteeuwen/go-bindata/...

mess:
	go build -o env/bin/mess github.com/natmeox/mess/cmd

assetsdev:
	env/bin/go-bindata -debug -o cmd/site.go $(ASSETS)

assetsprod:
	env/bin/go-bindata -o cmd/site.go $(ASSETS)
