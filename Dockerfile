# Build sidecar binary
FROM dockerhub.devops.telekom.de/golang:1.15.6 AS builder


COPY src /src
WORKDIR /src
# ENV GO111MODULE=on

RUN CGO_ENABLED=0 go build -a -tags netgo -ldflags '-w -extldflags "-static"'


# Run sidecar
FROM docker pull azul/zulu-openjdk

ENV SIDECAR=/usr/local/bin/vaultsidecar \
    USER_UID=1001 \
    USER_NAME=vaultsidecar \
    GV_ANSIBLE_VAULT_KEY="echo $GV_ANSIBLE_VAULT_KEY"

COPY --from=builder /src/vaultsidecar $SIDECAR
COPY vault /tmp/vault

#RUN /usr/local/bin/user_setup
ENTRYPOINT ["/usr/local/bin/vaultsidecar"]

