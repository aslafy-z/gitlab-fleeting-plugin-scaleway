FROM golang:1.24-alpine AS builder

WORKDIR /build

ADD go.mod go.sum /build/
RUN go mod download

ADD . /build/
RUN ls -al

RUN go build ./cmd/fleeting-plugin-scaleway

FROM gitlab/gitlab-runner:alpine-v18.0.2

COPY --from=builder /build/fleeting-plugin-scaleway /bin/fleeting-plugin-scaleway
