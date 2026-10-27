
dist/openapidocs:
	go build -o $@ ./cmd

.PHONY: clean
clean:
	rm -rf dist
