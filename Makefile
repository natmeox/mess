ASSETS=config.json.sample mess.sql static/ static/bootstrap/ static/bootstrap/mixins/ static/fonts/bootstrap/ static/javascripts/bootstrap/ template/

mess:
	go build -o env/bin/mess github.com/natmeox/mess/cmd

assetsdev:
	go-bindata -debug -o cmd/site.go $(ASSETS)

assetsprod:
	go-bindata -o cmd/site.go $(ASSETS)
