docker: compile
	docker build -t dasmith/e2pgs .

smalldocker: compile
	docker build -f Dockerfile.tiny -t dasmith/tinye2pgs .

compile:
	GOOS=linux GOARCH=386 CGO_ENABLED=0 go build -o e2pgs .
dependencies:
	go get -u github.com/aws/aws-sdk-go
	go get -u github.com/alecthomas/kingpin
	go get -u github.com/lib/pq
	go get -u github.com/nu7hatch/gouuid
	go get -u github.com/xtraclabs/pgeventstore
	go get -u github.com/xtraclabs/snspublish/db

