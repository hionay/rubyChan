IMAGE := hionayd/rubychan
TAG   := latest

.PHONY: build test clean run lint docker-build docker-push

build:
	CGO_ENABLED=0 go build -tags goolm -o rubyChan .

test:
	CGO_ENABLED=0 go test -tags goolm ./...

run:
	CGO_ENABLED=0 go run -tags goolm .

lint:
	CGO_ENABLED=0 go vet -tags goolm ./...

clean:
	rm -f rubyChan

docker-build:
	docker build -t $(IMAGE):$(TAG) .

docker-push: docker-build
	docker push $(IMAGE):$(TAG)
