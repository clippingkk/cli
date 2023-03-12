#!/bin/bash

echo "code sign started"

echo $MACOS_CERTIFICATE | base64 —decode > certificate.p12
security create-keychain -p $APPLE_DEVELOPER_CERT_PWD build.keychain
security default-keychain -s build.keychain
security unlock-keychain -p $APPLE_DEVELOPER_CERT_PWD build.keychain
security import certificate.p12 -k build.keychain -P $$APPLE_DEVELOPER_CERT_PWD -T /usr/bin/codesign
security set-key-partition-list -S apple-tool:,apple:,codesign: -s -k $APPLE_DEVELOPER_CERT_PWD build.keychain
/usr/bin/codesign --force -s $APPLE_DEVELOPER_CERT_PWD ./path/to/you/app -v

echo "code sign done..."