build:
	go build -v -i .
dev: build
	./bsalliance --local --log-level trace
clean:
	rm bsalliance.db
	rm bsalliance
