#!/bin/env bash
set -e

if [ ! -f "$spec_json_path" ] ; then
    echo "missing required input: spec_json_path"
    exit 1
fi

if [ -z "$algolia_app_id" ] ; then
    echo "missing required input: algolia_app_id"
    exit 1
fi

if [ -z "$algolia_api_key" ] ; then
    echo "missing required input: algolia_api_key"
    exit 1
fi

set -x

git clone https://github.com/bitrise-io/steplib-search.git "$BITRISE_SOURCE_DIR/steplib-search"

cd steplib-search

export BITRISE_SOURCE_DIR="$BITRISE_SOURCE_DIR/steplib-search"
export SPEC_JSON_PATH="$spec_json_path"
export ALGOLIA_APP_ID="$algolia_app_id"
export ALGOLIA_API_KEY="$algolia_api_key"
bitrise run update-algolia