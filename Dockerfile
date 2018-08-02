FROM golang:1.11beta2

CMD [ \
  "cd /tmp/cirrus-ci-build", \
  "ls", \
  "go version", \
  "go env", \
  "go vet ./...", \
  "go test -race ./...", \
  "go test -cover ./...", \
  "go test -run=NONE -bench . -benchmem ./...", \
]
