#!/bin/bash
# shellcheck disable=SC2046,SC2068,SC2181,SC2199
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# This script is designed to run specified linting/testing in Travis
# for modified files. Due to the multitude of different situations in
# Travis with comparing commits, linting/testing is performed only when
# reasonable and ideally only on modified code/files.


set -ex

MODIFIED_FILES=


if [[ -z "${TRAVIS_COMMIT_RANGE}" ]]; then
    # New branch, skip running lint tests
    echo "New branch detected, skipping lint tests."
    exit 0
else
    set +e
    # Attempt normal build
    MODIFIED_FILES="$(git diff --name-only --diff-filter=AM ${TRAVIS_COMMIT_RANGE} --)"
    if [[ $? -ne 0 ]]; then
        # A rebase or force push has been performed, best we can do is
        # check the most recent commit (as commit range now references a
        # bad commit; otherwise, no files will be collected and we "skip"
        MODIFIED_FILES="$(git diff --name-only --diff-filter=AM HEAD~1...HEAD --)"
    fi
    set -e
fi

function run_bashate {
    if [[ -n "${@}" ]]; then
        bashate -v --ignore E006 ${@}
    else
        echo -e "\033[0;32mNo shell changes."
    fi
}

function run_flake8 {
    if [[ -n "${@}" ]]; then
        flake8 ${@}
    else
        echo -e "\033[0;32mNo Python changes."
    fi
}

function run_go_test {
    # We attempt to find the set(s) of tests we run to only target
    # IBM Cloud code testing, so other provider test failures do block us
    find . -type d -name ibmcloud | sort | xargs -I % sh -c "go test %/..."
}

function run_golangci_lint {
    if [[ -n "${@}" ]]; then
        # We attempt to find the set of 'ibmcloud' code to run linting
        # on, as golangci-lint provides additional linting than golint
        # but requires special invocation to properly build and lint
        # so we cannot just check 'changed files'
        find . -type d -name ibmcloud | sort | xargs -I % sh -c "golangci-lint run --timeout 5m %/..."
    else
        echo -e "\033[0;32mNo Golang changes."
    fi
}

function run_golint {
    if [[ -n "${@}" ]]; then
        golint -set_exit_status ${@}
    else
        echo -e "\033[0;32mNo Golang changes."
    fi
}

function run_pylint {
    if [[ -n "${@}" ]]; then
        pylint ${@}
    else
        echo -e "\033[0;32mNo Python changes."
    fi
}

function run_shellcheck {
    if [[ -n "${@}" ]]; then
        shellcheck --exclude=SC2086 ${@}
    else
        echo -e "\033[0;32mNo shell changes."
    fi
}

function run_tflint {
    if [[ -n "${@}" ]]; then
        tflint ${@}
    else
        echo -e "\033[0;32mNo terraform changes."
    fi
}

function run_yamllint {
    if [[ -n "${@}" ]]; then
        yamllint ${@}
    else
        echo -e "\033[0;32mNo YAML changes."
    fi
}


for arg in "${@}"; do
    case ${arg} in
        bashate)
            run_bashate $(grep ".*\.sh$" <<< ${MODIFIED_FILES} | grep -v "vendor/*")
            ;;
        flake8)
            run_flake8 $(grep ".*\.py$" <<< ${MODIFIED_FILES} | grep -v "vendor/*")
            ;;
        golangci-lint)
            run_golangci_lint $(grep ".*\.go$" <<< ${MODIFIED_FILES} | grep -v "vendor/*")
            ;;
        golint)
            run_golint $(grep ".*\.go$" <<< ${MODIFIED_FILES} | grep -v "vendor/*")
            ;;
        go-test)
            # We run tests in specific modules, not based off changed files
            run_go_test
            ;;
        pylint)
            run_pylint $(grep ".*\.py$" <<< ${MODIFIED_FILES} | grep -v "vendor/*")
            ;;
        shellcheck)
            run_shellcheck $(grep ".*\.sh$" <<< ${MODIFIED_FILES} | grep -v "vendor/*")
            ;;
        tflint)
            run_tflint $(grep ".*\.tf$" <<< ${MODIFIED_FILES} | grep -v "vendor/*")
            ;;
        yamllint)
            run_yamllint $(grep ".*\.ya*ml$" <<< ${MODIFIED_FILES} | grep -v "vendor/*")
            ;;
        *)
            echo "Invalid lint check, skipping: ${arg}"
    esac
done

exit 0
