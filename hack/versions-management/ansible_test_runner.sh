#!/bin/bash
set -o errexit
set -o pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

source "$SCRIPT_DIR/github_auth.sh"

ARGS=()
if [ "${DRY_RUN:-false}" = "true" ]; then
  ARGS+=(--dry-run)
fi

python "$SCRIPT_DIR/ansible_test_runner.py" "${ARGS[@]}"

if [ "${DRY_RUN:-false}" = true ]; then
    echo "Ansible test runner has finished successfully"
else
    git remote add origin https://github.com/openshift-assisted/cluster-api-provider-openshift-assisted
    git remote set-url --push origin https://github.com/openshift-assisted/cluster-api-provider-openshift-assisted
    git add release-candidates.yaml
    git commit -m "Update release candidates status after testing" || echo "No changes to commit"
    git push origin HEAD:master
fi
