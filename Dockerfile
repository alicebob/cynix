FROM golang:1.20 as builder
WORKDIR /build
COPY go.mod go.sum main.go ./
RUN CGO_ENABLED=0 go build -o cynix

FROM nixos/nix
WORKDIR /
COPY --from=0 /build/cynix /
COPY /run.sh /
RUN	nix-channel --add https://nixos.org/channels/nixos-22.11 nixpkgs
RUN	nix-channel --update
# this might preinstall: RUN nix-build -A pythonFull '<nixpkgs>'
ENTRYPOINT /run.sh
