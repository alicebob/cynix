FROM golang:1.20 as builder
WORKDIR /build
COPY go.mod go.sum main.go ./
RUN go build -o cynix

FROM debian:stable-slim
RUN DEBIAN_FRONTEND=noninteractive apt update -y && apt -y install --no-install-recommends ca-certificates xz-utils curl

RUN install -d -m755 -o $(id -u) -g $(id -g) /nix
RUN curl -L https://nixos.org/nix/install | sh

WORKDIR /

RUN	nix-channel --add https://nixos.org/channels/nixos-22.11 nixpkgs
RUN	nix-channel --update
# this might preinstall: RUN nix-build -A pythonFull '<nixpkgs>'

COPY --from=0 /build/cynix /
COPY /run.sh /
ENTRYPOINT /run.sh
