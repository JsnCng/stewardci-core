#!/bin/bash

function die() {
    if [[ -n "$*" ]]; then
        echo "$@" >&2
    fi
    exit 1
}

PROJECT_ROOT=$(cd "$(dirname "$BASH_SOURCE")/.."; pwd)

if [[ -z $GOPATH ]]; then
    echo "error: GOPATH not set"
    exit 1
fi
GOPATH_1=${GOPATH%%:*}  # the first entry of the GOPATH

if [[ -z $CODEGEN_PKG ]]; then
    . ${PROJECT_ROOT}/hack/bootstrap-codegen.sh || die "Installation of code-generator failed"
fi
if [[ ! -f $CODEGEN_PKG/generate-groups.sh ]]; then
    echo "error: CODEGEN_PKG does not point to a directory containing 'generate-groups.sh': $CODEGEN_PKG"
    exit 1
fi

if [[ ! -x "$GOPATH_1/bin/mockgen" ]]; then
    go get github.com/golang/mock/mockgen || die "Installation of mockgen failed"
fi

if [[ "$1" == "--verify" || "$1" == "-v" ]]; then
    VERIFY=true
else    
    VERIFY=false
fi

PROJECT_ROOT=$(cd "$(dirname "$BASH_SOURCE")/.."; pwd)
GEN_DIR="$PROJECT_ROOT/gen"

echo
echo "PROJECT_ROOT: $PROJECT_ROOT"
echo "GEN_DIR:      $GEN_DIR"
echo "CODEGEN_PKG:  $CODEGEN_PKG"
echo "GOPATH:       $GOPATH_1"
echo "VERIFY:       $VERIFY"

echo
echo "## Cleanup old generated stuff ####################"
if [ "$VERIFY" = true ]; then
    set -x
    rm -rf \
        "${GEN_DIR}" \
        "${GOPATH_1}/bin/"{client-gen,deepcopy-gen,defaulter-gen,informer-gen,lister-gen}
    set +x
else
    set -x
    rm -rf \
        "${PROJECT_ROOT}/pkg/client" \
        "${PROJECT_ROOT}/pkg/tektonclient" \
        "${PROJECT_ROOT}/pkg/apis/steward/v1alpha1/zz_generated.deepcopy.go" \
        "${PROJECT_ROOT}/pkg/k8s/mocks/mocks.go" \
        "${GEN_DIR}/github.com" \
        "${GOPATH_1}/bin/"{client-gen,deepcopy-gen,defaulter-gen,informer-gen,lister-gen}
    set +x
fi

echo
echo "## Generate #######################################"
set -x
"${CODEGEN_PKG}/generate-groups.sh" \
    all \
    github.com/SAP/stewardci-core/pkg/client \
    github.com/SAP/stewardci-core/pkg/apis \
    steward:v1alpha1 \
    --go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt" \
    --output-base "${GEN_DIR}"
set +x
set -x
"${CODEGEN_PKG}/generate-groups.sh" \
    "client,informer,lister" \
    github.com/SAP/stewardci-core/pkg/tektonclient \
    github.com/tektoncd/pipeline/pkg/apis \
    pipeline:v1alpha1 \
    --go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt" \
    --output-base "${GEN_DIR}"
set +x

if [ "$VERIFY" = true ]; then
    echo
    echo "## Verifying generated sources ####################"
    set -x
    diff -Naupr ${GEN_DIR}/github.com/SAP/stewardci-core/pkg/client/ ${PROJECT_ROOT}/pkg/client/ || die "Regeneration required for clients"
    diff -Naupr ${GEN_DIR}/github.com/SAP/stewardci-core/pkg/tektonclient/ ${PROJECT_ROOT}/pkg/tektonclient/ || die "Regeneration required for tektonclients"
    diff -Naupr ${GEN_DIR}/github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1/zz_generated.deepcopy.go ${PROJECT_ROOT}/pkg/apis/steward/v1alpha1/zz_generated.deepcopy.go || die "Regeneration required for apis"
    set +x
else
    echo
    echo "## Move generated files ###########################"
    set -x
    mv "${GEN_DIR}/github.com/SAP/stewardci-core/pkg/client" "${PROJECT_ROOT}/pkg/"
    mv "${GEN_DIR}/github.com/SAP/stewardci-core/pkg/tektonclient" "${PROJECT_ROOT}/pkg/"
    cp -r "${GEN_DIR}/github.com/SAP/stewardci-core/pkg/apis" "${PROJECT_ROOT}/pkg/"
    rm -rf "${GEN_DIR}/github.com"
    set +x
fi


echo
if [ "$VERIFY" = true ]; then
    echo "## Verify mocks for package 'k8s' ###############"
    set -x
    "$GOPATH_1/bin/mockgen" \
        -copyright_file="${PROJECT_ROOT}/hack/boilerplate.go.txt" \
        -destination="${GEN_DIR}/pkg/k8s/mocks/mocks.go" \
        -package=mocks \
        github.com/SAP/stewardci-core/pkg/k8s \
        PipelineRun,ClientFactory,PipelineRunFetcher,SecretProvider,NamespaceManager
    diff -Naupr ${GEN_DIR}/pkg/k8s/mocks/mocks.go ${PROJECT_ROOT}/pkg/k8s/mocks/mocks.go || die "Regeneration required for apis"
    set +x
else
    echo "## Generate mocks for package 'k8s' ###############"
    set -x
    "$GOPATH_1/bin/mockgen" \
        -copyright_file="${PROJECT_ROOT}/hack/boilerplate.go.txt" \
        -destination="${PROJECT_ROOT}/pkg/k8s/mocks/mocks.go" \
        -package=mocks \
        github.com/SAP/stewardci-core/pkg/k8s \
        PipelineRun,ClientFactory,PipelineRunFetcher,SecretProvider,NamespaceManager
    set +x
fi