FROM golang:1.11beta2

RUN mkdir -p /xmpp
COPY * /xmpp/

CMD [ \
  "go version", \
  "go env", \
  "go vet ./...", \
  "go test -race ./...", \
  "go test -cover ./...", \
  "go test -run=NONE -bench . -benchmem ./...", \
]
