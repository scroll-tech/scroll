#!/bin/bash

${GOROOT}/bin/bin/gocover-cobertura < coverage.bridge.txt > coverage.bridge.xml
${GOROOT}/bin/bin/gocover-cobertura < coverage.db.txt > coverage.db.xml
${GOROOT}/bin/bin/gocover-cobertura < coverage.common.txt > coverage.common.xml
${GOROOT}/bin/bin/gocover-cobertura < coverage.coordinator.txt > coverage.coordinator.xml
npx cobertura-merge -o cobertura.xml package1=coverage.bridge.xml package2=coverage.db.xml package3=coverage.common.xml package4=coverage.coordinator.xml