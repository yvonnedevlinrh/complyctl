#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# How to run the script
# Assume that you have already installed a fresh RHEL
# 1. Download/copy this script in your RHEL system
# 2. Run the script
# chmod +x quick_start.sh
# export RHEL_APPS_REPO=$RHEL_APPS_REPO
# export COMPLYTIME_DEV_MODE=1
# sh quick_start.sh

set +e
# Check if the scap-security-guide package is available in the enabled repositories
if ! dnf provides scap-security-guide; then
    echo "No working repository is available to install scap-security-guide."

    # Check if RHEL_APPS_REPO variable is set
    if [ -z "$RHEL_APPS_REPO" ]; then
        echo "Error: RHEL_APPS_REPO is not set. Please set the variable and try again."
        exit 1
    else
        echo "Setting up CaC Apps repository..."
        cat > /etc/yum.repos.d/cac.repo <<EOF
[cac_apps_repo]
name=CaC Apps Repo
baseurl=${RHEL_APPS_REPO}
enabled=1
gpgcheck=0
EOF
        echo "CaC Apps repository has been added."
    fi
fi

echo "Installing dependencies..."
dnf update -y
dnf install git wget make scap-security-guide -y
rm -rf /usr/bin/go
go_mod="https://raw.githubusercontent.com/complytime/complyctl/main/go.mod"
go_version=$(curl -s "$go_mod" | grep '^go' | awk '{print $2}')
go_tar_file="go${go_version}.linux-amd64.tar.gz"
wget "https://go.dev/dl/$go_tar_file"
tar -C /usr/local -xvzf "$go_tar_file"
rm -rf "$go_tar_file"
export PATH="$PATH:/usr/local/go/bin"
# Install and build complyctl
echo "Cloning the complyctl repository..."
complyctlrepo="${REPO:-https://github.com/complytime/complyctl}"
complyctlbranch="${BRANCH:-main}"
git clone -b "${complyctlbranch}" "${complyctlrepo}"
cd complyctl && make build && cp ./bin/complyctl /usr/local/bin
echo "complyctl installed successfully!"
# Run complyctl list to create the workspace
set +e
# Running list command that will fail due to no requirements files
echo "Attempting to run the command complyctl list."
bin/complyctl list 2>/dev/null
echo "An error occurred, but script continues after the complyctl list."
# Copy the artifacts to workspace
cp docs/samples/sample-component-definition.json ~/.local/share/complytime/bundles
cp docs/samples/sample-profile.json docs/samples/sample-catalog.json ~/.local/share/complytime/controls

# Copy the binary plugin and manifest files
cp -rp bin/openscap-plugin ~/.local/share/complytime/plugins
checksum=$(sha256sum ~/.local/share/complytime/plugins/openscap-plugin | cut -d ' ' -f 1)
cat > ~/.local/share/complytime/plugins/c2p-openscap-manifest.json << EOF
{
  "metadata": {
    "id": "openscap",
    "description": "My openscap plugin",
    "version": "0.0.1",
    "types": [
      "pvp"
    ]
  },
  "executablePath": "openscap-plugin",
  "sha256": "$checksum",
  "configuration": [
    {
      "name": "workspace",
      "description": "Directory for writing plugin artifacts",
      "required": true
    },
    {
      "name": "profile",
      "description": "The OpenSCAP profile to run for assessment",
      "required": true
    },
    {
      "name": "datastream",
      "description": "The OpenSCAP datastream to use. If not set, the plugin will try to determine it based on system information",
      "required": false
    },
    {
      "name": "policy",
      "description": "The name of the generated tailoring file",
      "default": "tailoring_policy.xml",
      "required": false
    },
    {
      "name": "arf",
      "description": "The name of the generated ARF file",
      "default": "arf.xml",
      "required": false
    },
    {
      "name": "results",
      "description": "The name of the generated results file",
      "default": "results.xml",
      "required": false
    }
  ]
}
EOF
