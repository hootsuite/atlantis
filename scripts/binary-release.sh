#!/bin/bash

# define architecture we want to build
XC_ARCH=${XC_ARCH:-"386 amd64 arm"}
XC_OS=${XC_OS:-linux darwin}
XC_EXCLUDE_OSARCH="!darwin/arm !darwin/386"

# clean up
echo "-> running clean up...."
rm -rf output/*

if ! which gox > /dev/null; then
    echo "-> installing gox..."
    go get -u github.com/mitchellh/gox
fi

# build
# we want to build statically linked binaries
export CGO_ENABLED=0
echo "-> building..."
gox \
    -os="${XC_OS}" \
    -arch="${XC_ARCH}" \
    -osarch="${XC_EXCLUDE_OSARCH}" \
    -output "output/{{.OS}}_{{.Arch}}/atlantis" \
    .

# Zip and copy to the dist dir
echo ""
echo "Packaging..."
for PLATFORM in $(find ./output -mindepth 1 -maxdepth 1 -type d); do
    OSARCH=$(basename ${PLATFORM})
    echo "--> ${OSARCH}"

    pushd $PLATFORM >/dev/null 2>&1
    zip ../atlantis_${OSARCH}.zip ./*
    popd >/dev/null 2>&1
done

echo ""
echo ""
echo "-----------------------------------"
echo "Output:"
ls -alh output/