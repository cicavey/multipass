multipass: main.go
	@CGO_ENABLED=0 go build -a
docker: multipass
	docker build -t multipass .
clean:
	@rm -f multipass
.PHONY: clean
