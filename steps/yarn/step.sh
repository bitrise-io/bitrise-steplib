#!/bin/bash

set -e

if [ ! -z "${workdir}" ] ; then
  echo "==> Switching to working directory: ${workdir}"
  cd "${workdir}"
  if [ $? -ne 0 ] ; then
    echo " [!] Failed to switch to working directory: ${workdir}"
    exit 1
  fi
fi

yarn ${command} ${options}
