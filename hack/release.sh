#!/usr/bin/env bash
source .env

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

readonly LOCALBIN=${LOCALBIN:-$(pwd)/bin}
readonly KUSTOMIZE=${KUSTOMIZE:-$LOCALBIN/kustomize}
readonly GORELEASER_VERSION="${GORELEASER_VERSION:-$ENV_GORELEASER_VERSION}"
readonly IMAGE="${IMAGE:-$ENV_IMAGE}"
readonly IMAGE_EXPERIMENTAL=${IMAGE}-experimental
readonly CURRENT_VERSION="$1"

function prepare_release_artefacts() {
  echo "Preparing release artefacts"
  cd config/manager && ${KUSTOMIZE} edit set image controller="${IMAGE}" && cd ../..
  # Create the resources file that is used for creating the ModuleTemplate for regular
  ${KUSTOMIZE} build config/default >telemetry-manager.yaml
  # Create the resources file that is used for creating the ModuleTemplate for experimental release
  cd config/manager && ${KUSTOMIZE} edit set image controller="${IMAGE_EXPERIMENTAL}" && cd ../..
  ${KUSTOMIZE} build config/development >telemetry-manager-experimental.yaml
  # Rename the file for Telemetry default CR to have a better naming as a release artefact
  cp ./config/samples/operator_v1alpha1_telemetry.yaml telemetry-default-cr.yaml
}

get_previous_release_version() {
  # get list of tags in a reverse chronological order excluding dev tags,
  # sort them based on major, minor, patch numerically, grab the first release before the current one
  TAG_LIST_WITH_PATCH=$(git tag --sort=-creatordate | grep -E "^[0-9]+.[0-9]+.[0-9]$" | sort -t "." -k1,1n -k2,2n -k3,3n | grep -B 1 "${CURRENT_VERSION}" | head -1)
  export GORELEASER_PREVIOUS_TAG=${TAG_LIST_WITH_PATCH}
}

function create_github_release() {
  echo "Creating the Github release"
  git reset --hard
  curl -sL https://git.io/goreleaser | VERSION=${GORELEASER_VERSION} bash
}

function main() {
  prepare_release_artefacts
  get_previous_release_version
  create_github_release
}

main
