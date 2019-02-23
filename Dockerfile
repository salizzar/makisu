FROM golang:1.11 AS builder

RUN mkdir -p /go/src/github.com/uber/makisu
WORKDIR /go/src/github.com/uber/makisu

ADD Makefile .
RUN make ext-tools/Linux/dep

ADD Gopkg.toml Gopkg.lock ./
ADD .git ./.git
ADD bin ./bin
ADD lib ./lib
RUN make lbins


FROM golang:1.11 AS gcr_cred_helper_builder
ADD Makefile .
RUN make gcr-helper


FROM golang:1.11 AS ecr_cred_helper_builder
ADD Makefile .
RUN make ecr-helper


FROM scratch
COPY --from=builder                 /go/bin/makisu.linux                /makisu-internal/makisu
COPY --from=gcr_cred_helper_builder /go/bin/docker-credential-gcr       /makisu-internal/docker-credential-gcr
COPY --from=ecr_cred_helper_builder /go/bin/docker-credential-ecr-login /makisu-internal/docker-credential-ecr-login
ADD ./assets/cacerts.pem /makisu-internal/certs/cacerts.pem

ENTRYPOINT ["/makisu-internal/makisu"]
