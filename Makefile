debug:
	go run -mod vendor cmd/server/main.go -nextzen-apikey $(APIKEY) -enable-placeholder -placeholder-endpoint $(SEARCH) -enable-oembed -oembed-endpoints 'https://millsfield.sfomuseum.org/oembed/' -enable-writer -writer-uri 'whosonfirst://?writer=stdout://&reader=fs:///usr/local/data/sfomuseum-data-collection/data'
