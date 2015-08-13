#!/bin/bash

THIS_SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

set -e

if [ -z "${AWS_ACCESS_KEY_ID}" ] ; then
	echo " [!] No AWS_ACCESS_KEY_ID specified"
	exit 1
fi
if [ -z "${AWS_SECRET_ACCESS_KEY}" ] ; then
	echo " [!] No AWS_ACCESS_KEY_ID specified"
	exit 1
fi
if [ -z "${S3_UPLOAD_BUCKET}" ] ; then
	echo " [!] No AWS_ACCESS_KEY_ID specified"
	exit 1
fi

set -v

# normalized repo_root_dir
repo_root_dir="${THIS_SCRIPT_DIR}/../.."
cd "${repo_root_dir}"
repo_root_dir="$(pwd)"
echo "repo_root_dir: ${repo_root_dir}"
# other paths
tmp_work_dir="${repo_root_dir}/_tmp"
steps_dir_path="${repo_root_dir}/steps"


mkdir -p "${tmp_work_dir}"
cd "${tmp_work_dir}"

# export GOPATH="${tmp_work_dir}/go"
# mkdir -p "${GOPATH}/src"
# go get gopkg.in/yaml.v2

go run "${THIS_SCRIPT_DIR}/deploy_to_s3.go" \
	-accesskey "${AWS_ACCESS_KEY_ID}" \
	-secretkey "${AWS_SECRET_ACCESS_KEY}" \
	-s3bucket "${S3_UPLOAD_BUCKET}" \
	-s3bucket-region "${S3_UPLOAD_BUCKET_REGION}" \
	-stepsdir "${steps_dir_path}" \
	-collection-root-dir "${repo_root_dir}" \
	-collection-spec-json "${SPEC_JSON_PATH}"
