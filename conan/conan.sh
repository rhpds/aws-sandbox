#!/bin/bash

set -u -o pipefail

##############
# conf
##############

# Number of aws-nuke processes to run in parallel
threads="${threads:-12}"
# Number of attempts to run cleanup on a sandbox
max_retries="${max_retries:-2}"
aws_nuke_retries=${aws_nuke_retries:-0}

# AWS profile
aws_profile="${aws_profile:-pool-manager}"

# DynamoDB
dynamodb_profile="${dynamodb_profile:-pool-manager}"
dynamodb_table="${dynamodb_table:-accounts}"
dynamodb_region="${dynamodb_region:-us-east-1}"

# Pause between each iteration that gets the list of sandboxes to cleanup
poll_interval="${poll_interval:-60}"

# aws-nuke path
aws_nuke_binary_path="${aws_nuke_binary_path:-aws-nuke}"

# Noop: don't actually touch the sandboxes
noop=${noop:-false}

# python virtualenv
VENV=${VENV:-~/pool_management/python_virtualenv}
NOVENV=${NOVENV:-false}

# AWS CLI
AWSCLI=${AWSCLI:-~/pool_management/aws/dist/aws}

# Conan instance: the name of the host running the cleanup
conan_instance=${conan_instance:-$(hostname)}

# Workdir
workdir=${workdir:-~/pool_management}

# Vault file
vault_file=${vault_file:-~/secrets/infra-sandbox-vault}

# Kerberos
kerberos_keytab=${kerberos_keytab:-~/secrets/hostadmin.keytab}
kerberos_user=${kerberos_user:-hostadmin}
kerberos_password=${kerberos_password:-}

# Pattern to filter the sandboxes to cleanup
sandbox_filter=${sandbox_filter:-}

if [ -n "${kerberos_password}" ]; then
    unset kerberos_keytab
fi

# Lock timeout:  the number of hours after which a lock on a sandbox expires.
# For ex: '2': a conan process will have 2h to cleanup the sandbox before another
# process can claim the sandbox for cleanup.
# That parameter prevent a sandbox from being locked forever if something goes wrong with
# the conan process owning the lock.
lock_timeout=${lock_timeout:-2}

# Lock timeout in minutes to add to the previous lock_timeout
lock_timeout_m=${lock_timeout_m:-0}


# Variable to manage output loglevel
debug=false

##############

export AWSCLI
export NOVENV
export VENV
export aws_nuke_binary_path
export aws_profile
export conan_instance
export dynamodb_profile
export dynamodb_region
export dynamodb_table
export kerberos_keytab
export kerberos_password
export kerberos_user
export lock_timeout
export lock_timeout_m
export max_retries
export aws_nuke_retries
export noop
export poll_interval
export threads
export vault_file
export workdir
export sandbox_filter
export debug

ORIG="$(cd "$(dirname "$0")" || exit; pwd)"


prepare_workdir() {
    mkdir -p "${workdir}"

    if [ "${NOVENV}" = true ]; then
        return
    fi

    if [ ! -d "${VENV}" ]; then
        set -e
        echo "Create python virtualenv"
        python3 -mvenv "${VENV}"
        # shellcheck source=/dev/null
        . "${VENV}/bin/activate"
        pip install --upgrade pip
        pip install -r "${ORIG}/../playbooks/requirements.txt"
        set +e
    fi
    # shellcheck source=/dev/null
    . "$VENV/bin/activate"
}


pre_checks() {
    for c in sandbox-list \
             rush \
             kinit; do
        if ! command -v $c &>/dev/null; then
            echo "'${c}' command not found" >&2
            sync
            exit 5
        fi
    done
    if ! AWS_PROFILE=${dynamodb_profile} \
        AWS_REGION=${dynamodb_region} \
        dynamodb_table=${dynamodb_table} \
        sandbox-list --to-cleanup --no-headers &> /dev/null
    then
        echo "command failed: sandbox-list --to-cleanup"
        sync
        exit 5
    fi
}

echo "AWS profile: ${aws_profile}"
echo "DynamoDB profile: ${dynamodb_profile}"
echo "DynamoDB table: ${dynamodb_table}"

pre_checks
prepare_workdir

cd "${ORIG}"

while true; do

    (
        export AWS_PROFILE=${dynamodb_profile}
        export AWS_REGION=${dynamodb_region}
        export dynamodb_table=${dynamodb_table}
        sandbox-list --to-cleanup --no-headers
    ) \
        | grep -E "${sandbox_filter}" \
        | rush --immediate-output -j "${threads}" './wipe_sandbox.sh {1}'

    sleep "${poll_interval}"
done
