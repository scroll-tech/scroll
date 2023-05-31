#!/bin/bash
set -uex
gocover-cobertura < coverage.bridge.txt > coverage.bridge.xml
gocover-cobertura < coverage.db.txt > coverage.db.xml
gocover-cobertura < coverage.common.txt > coverage.common.xml
gocover-cobertura < coverage.coordinator.txt > coverage.coordinator.xml
#${GOROOT}/bin/bin/gocover-cobertura < coverage.integration.txt > coverage.integration.xml

npx cobertura-merge -o cobertura.xml \
    package1=coverage.bridge.xml \
    package2=coverage.db.xml \
    package3=coverage.common.xml \
    package4=coverage.coordinator.xml
    # package5=coverage.integration.xml
