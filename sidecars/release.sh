#!/usr/bin/env bash
# This file is part of MinIO DirectPV
# Copyright (c) 2024 MinIO, Inc.
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.

function generate_dockerfile() {
    cat >Dockerfile.minio <<EOF
FROM registry.access.redhat.com/ubi8/ubi-micro:8.10
LABEL maintainers="dev@min.io"
LABEL description="${PROJECT_DESCRIPTION}"
COPY ${PROJECT_NAME} /${PROJECT_NAME}
COPY LICENSE /licenses/LICENSE
ENTRYPOINT ["/${PROJECT_NAME}"]
EOF
}

function generate_goreleaser_yaml() {
    cat >.goreleaser.yml <<EOF
version: 2

project_name: ${PROJECT_NAME}

release:
   name_template: "Release version {{.Version}}"

   github:
    owner: minio
    name: "{{ .ProjectName }}"

   extra_files:
     - glob: "*.minisig"
     - glob: "*.zip"

before:
  hooks:
    - go mod tidy -compat=1.22
    - go mod download

builds:
  -
    main: ./cmd/{{ .ProjectName }}
    id: "{{ .ProjectName }}"
    binary: "{{ .ProjectName }}"
    goos:
      - linux
    goarch:
      - amd64
      - arm64
      - ppc64le
    env:
      - CGO_ENABLED=0
    flags:
      - -trimpath
      - -tags="osusergo netgo static_build"
    ldflags:
      - -s -w -X main.Version={{ .Tag }}

archives:
  -
    allow_different_binary_count: true
    format: binary

changelog:
  sort: asc

dockers:
- image_templates:
  - "quay.io/minio/{{ .ProjectName }}:{{ .Tag }}-amd64"
  use: buildx
  goarch: amd64
  ids:
    - "{{ .ProjectName }}"
  dockerfile: Dockerfile.minio
  extra_files:
    - LICENSE
  build_flag_templates:
  - "--platform=linux/amd64"
- image_templates:
  - "quay.io/minio/{{ .ProjectName }}:{{ .Tag }}-ppc64le"
  use: buildx
  goarch: ppc64le
  ids:
    - "{{ .ProjectName }}"
  dockerfile: Dockerfile.minio
  extra_files:
    - LICENSE
  build_flag_templates:
  - "--platform=linux/ppc64le"
- image_templates:
  - "quay.io/minio/{{ .ProjectName }}:{{ .Tag }}-arm64"
  use: buildx
  goarch: arm64
  ids:
    - "{{ .ProjectName }}"
  dockerfile: Dockerfile.minio
  extra_files:
    - LICENSE
  build_flag_templates:
  - "--platform=linux/arm64"
docker_manifests:
- name_template: quay.io/minio/{{ .ProjectName }}:{{ .Tag }}
  image_templates:
  - quay.io/minio/{{ .ProjectName }}:{{ .Tag }}-amd64
  - quay.io/minio/{{ .ProjectName }}:{{ .Tag }}-arm64
  - quay.io/minio/{{ .ProjectName }}:{{ .Tag }}-ppc64le
- name_template: quay.io/minio/{{ .ProjectName }}:latest
  image_templates:
  - quay.io/minio/{{ .ProjectName }}:{{ .Tag }}-amd64
  - quay.io/minio/{{ .ProjectName }}:{{ .Tag }}-arm64
  - quay.io/minio/{{ .ProjectName }}:{{ .Tag }}-ppc64le
EOF
}

# usage: release <TAG>
function release() {
    if [ -z "${GITHUB_PROJECT_NAME}" ]; then
        echo "GITHUB_PROJECT_NAME environment variable must be set"
        exit 255
    fi

    if [ -z "${PROJECT_NAME}" ]; then
        echo "PROJECT_NAME environment variable must be set"
        exit 255
    fi

    if [ -z "${PROJECT_DESCRIPTION}" ]; then
        echo "PROJECT_DESCRIPTION environment variable must be set"
        exit 255
    fi

    if [ "$#" -ne 1 ]; then
        echo "VERSION tag must be passed"
        exit 255
    fi

    set -o errexit
    set -o nounset
    set -o pipefail

    tag="$1"
    if ! curl -sfL "https://api.github.com/repos/kubernetes-csi/${GITHUB_PROJECT_NAME}/releases" | jq -r ".[].tag_name | select(. == \"${tag}\")" | grep -q .; then
        echo "ERROR: build tag ${tag} not found in https://github.com/kubernetes-csi/${GITHUB_PROJECT_NAME}.git"
        exit 1
    fi

    last_tag=$(curl -sfL "https://quay.io/api/v1/repository/minio/${PROJECT_NAME}" | jq -r ".tags | keys_unsorted[] | select(startswith(\"${tag}-\"))" | sort --reverse --version-sort | head -n1)
    if [ -z "${last_tag}" ]; then
        prev_tag=$(curl -sfL "https://quay.io/api/v1/repository/minio/${PROJECT_NAME}" | jq -r '.tags | keys_unsorted[]' | sort --reverse --version-sort | head -n1)
        if [ "${tag}" == "${prev_tag}" ]; then
            curr_tag="${tag}-1"
        else
            curr_tag="${tag}-0"
        fi
    else
        prev_tag="${last_tag}"
        build=${last_tag/"$tag"-/}
        (( build = build + 1 ))
        curr_tag="${tag}-${build}"
    fi

    mkdir -p "${GOPATH}/src/github.com/kubernetes-csi"
    cd "${GOPATH}/src/github.com/kubernetes-csi"
    git clone "https://github.com/kubernetes-csi/${GITHUB_PROJECT_NAME}.git"
    cd "${GITHUB_PROJECT_NAME}"
    git checkout -b "tag-${tag}" "${tag}"
    git tag "${curr_tag}" "${tag}"

    generate_dockerfile
    generate_goreleaser_yaml

    export GORELEASER_CURRENT_TAG="${curr_tag}"
    export GORELEASER_PREVIOUS_TAG="${prev_tag}"
    if [ -z "${DRY_RUN}" ]; then
        goreleaser release --snapshot --clean
    else
        goreleaser release
    fi
}
