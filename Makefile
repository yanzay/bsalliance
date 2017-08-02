build:
	go build -v -i .
dev: build
	./bsalliance --eng --log-level trace
clean:
	rm bsalliance.db
	rm bsalliance
