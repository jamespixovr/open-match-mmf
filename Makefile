build:
	go build cmd/main.go

docker:
	docker build -t jarmex/open-match-mmf .

push:
	docker push jarmex/open-match-mmf
