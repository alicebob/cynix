FROM golang:1.20 as builder
WORKDIR /build
COPY go.mod go.sum main.go ./
RUN go build -o cynix

FROM debian:stable-slim
RUN DEBIAN_FRONTEND=noninteractive apt update -y && apt -y install --no-install-recommends ca-certificates xz-utils curl \
	libkrb5-3 zlib1g liblttng-ust0 libssl1.1 libicu67

RUN mkdir -p /etc/nix
RUN chmod 0755 /etc/nix
RUN echo "build-users-group =" >> /etc/nix/nix.conf
RUN echo "experimental-features = nix-command flakes" >> /etc/nix/nix.conf

## part of this comes from https://aaronlevin.ca/post/100703631408/installing-nix-within-a-docker-container
RUN adduser --disabled-password --gecos '' cynix
RUN mkdir -m 0755 /nix && chown cynix /nix
USER cynix
ENV USER cynix
WORKDIR /home/cynix

RUN curl -L https://nixos.org/nix/install | sh -s -- --no-daemon

RUN	. .nix-profile/etc/profile.d/nix.sh && nix-channel --add https://nixos.org/channels/nixos-22.11 nixpkgs
RUN	. .nix-profile/etc/profile.d/nix.sh && nix-channel --update
# this might preinstall: RUN nix-build -A pythonFull '<nixpkgs>'

COPY --from=0 /build/cynix .
COPY /run.sh .
ENTRYPOINT /home/cynix/run.sh
