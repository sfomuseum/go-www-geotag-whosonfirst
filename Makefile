debug:
	go run -mod vendor cmd/server/main.go -nextzen-apikey $(APIKEY) -enable-placeholder -placeholder-endpoint $(SEARCH) -enable-oembed -oembed-endpoints 'https://millsfield.sfomuseum.org/oembed/?url={url}' -enable-writer -writer-uri 'whosonfirst://?writer=$(WRITER)&reader=$(READER)&update=1&source=sfomuseum'
