#!/bin/bash
# Integration tests that create a full cluster.

set -exo pipefail

cd $(dirname $(dirname $(realpath $0)))
make install
test/k3d-cluster-network/e2e.sh
test/kind-cluster-network/e2e.sh
test/minikube-cluster-network/e2e.sh
