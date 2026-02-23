#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Initializes the complyctl environment for e2e and integration tests.
#
# This script performs the following setup tasks:
#  - Build complyctl and initializes the complytime workspace.
#  - Downloads the chosen product's controls and component definitions.
#  - Sets up necessary plugins.
#
# Usage:
#   sh build_init_env.sh $product $catalog $profile
#   sh build_init_env.sh fedora cusp_fedora fedora-cusp_fedora-default

URL="https://raw.githubusercontent.com/ComplianceAsCode/oscal-content/refs/heads/main/"
WDIR=".local/share/complytime"

product=$1
catalog=$2
profile=$3

if [ "$#" -lt 3 ]; then
    echo "Please provide the necessary inputs."
    exit 1
fi

# Build the compyctl
make build && cp ./bin/complyctl /usr/local/bin
# Running list command that will fail due to no requirements files
echo "Attempting to run the command complyctl list."
set +e
complyctl list 2>/dev/null
echo "The error is expected because there is no content, this will create needed directoris for further test."
# Download OSCAL content
wget "$URL/profiles/$3/profile.json" -O "$HOME/$WDIR/controls/profile.json"
wget "$URL/catalogs/$2/catalog.json" -O "$HOME/$WDIR/controls/catalog.json"
wget "$URL/component-definitions/$1/$3/component-definition.json" -O "$HOME/$WDIR/bundles/component-definition.json"
# Update trestle path
sed -i "s|trestle://catalogs/$2/catalog.json|trestle://controls/catalog.json|" "$HOME/$WDIR/controls/profile.json"
sed -i "s|trestle://profiles/$3/profile.json|trestle://controls/profile.json|" "$HOME/$WDIR/bundles/component-definition.json"
# Setup plugin
cp -rp bin/openscap-plugin "$HOME/$WDIR/plugins"
checksum=$(sha256sum "$HOME/$WDIR/plugins/openscap-plugin" | cut -d " " -f 1)
jq --arg new_sum "$checksum" '.sha256 = $new_sum' "docs/samples/c2p-openscap-manifest.json" > "docs/samples/c2p-openscap-manifest.json.tmp"
mv docs/samples/c2p-openscap-manifest.json.tmp "$HOME/$WDIR/plugins/c2p-openscap-manifest.json"
echo "Build and init finished."
