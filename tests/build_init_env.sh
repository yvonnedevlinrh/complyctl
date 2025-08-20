#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Initializes the complyctl environment for e2e and integration tests.
#
# This script performs the following setup tasks:
#  - Build complyctl and initializes the complytime workspace.
#  - Downloads required controls and component definitions.
#  - Sets up necessary plugins.
#
# Usage:
#   sh build_init_env.sh

URL="https://raw.githubusercontent.com/ComplianceAsCode/oscal-content/refs/heads/main/"
CATALOG="cusp_fedora"
PROFILE="fedora-cusp_fedora-default"
PRODUCT="fedora"
WDIR=".local/share/complytime"

# Build the compyctl
make build && cp ./bin/complyctl /usr/local/bin
# Running list command that will fail due to no requirements files
echo "Attempting to run the command complyctl list."
set +e
complyctl list 2>/dev/null
echo "An error occurred, but script continues after the complyctl list."
# Download fedora cusp OSCAL content
wget $URL/profiles/$PROFILE/profile.json -O $HOME/$WDIR/controls/profile.json
wget $URL/catalogs/$CATALOG/catalog.json -O $HOME/$WDIR/controls/catalog.json
wget $URL/component-definitions/$PRODUCT/$PROFILE/component-definition.json -O $HOME/$WDIR/bundles/component-definition.json
# Update trestle path
sed -i "s|trestle://catalogs/$CATALOG/catalog.json|trestle://controls/catalog.json|" $HOME/$WDIR/controls/profile.json
sed -i "s|trestle://profiles/$PROFILE/profile.json|trestle://controls/profile.json|" $HOME/$WDIR/bundles/component-definition.json
# Setup plugin
cp -rp bin/openscap-plugin $HOME/$WDIR/plugins
checksum=$(sha256sum $HOME/$WDIR/plugins/openscap-plugin| cut -d " " -f 1 )
sed -i "s/checksum_placeholder/$checksum/" docs/samples/c2p-openscap-manifest.json
cp docs/samples/c2p-openscap-manifest.json $HOME/$WDIR/plugins
echo "Build and init finished."
