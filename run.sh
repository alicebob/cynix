#!/bin/sh
. .nix-profile/etc/profile.d/nix.sh
./cynix -pat="${CYNIX_PAT}" -owner="${CYNIX_OWNER}" -repo="${CYNIX_REPO}" -name=cynix1
