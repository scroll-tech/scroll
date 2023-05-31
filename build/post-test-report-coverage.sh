#!/bin/bash
set -uex
~/go/bin/gocover-cobertura  < coverage.bridge.txt > coverage.bridge.xml
~/go/bin/gocover-cobertura  < coverage.db.txt > coverage.db.xml
~/go/bin/gocover-cobertura  < coverage.common.txt > coverage.common.xml
~/go/bin/gocover-cobertura  < coverage.coordinator.txt > coverage.coordinator.xml
#${GOROOT}/bin/bin/gocover-cobertura < coverage.integration.txt > coverage.integration.xml

npx cobertura-merge -o cobertura.xml \
    package1=coverage.bridge.xml \
    package2=coverage.db.xml \
    package3=coverage.common.xml \
    package4=coverage.coordinator.xml
    # package5=coverage.integration.xml
