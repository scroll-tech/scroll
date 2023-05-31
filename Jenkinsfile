imagePrefix = 'scrolltech'
credentialDocker = 'dockerhub'

pipeline {
    agent any
    options {
        timeout (20)
    }
    tools {
        nodejs "nodejs"
        go 'go-1.18'
    }
    environment {
        GOBIN = '/home/ubuntu/go/bin/'
        GO111MODULE = 'on'
        PATH="/home/ubuntu/.cargo/bin:$PATH"
        LD_LIBRARY_PATH="$LD_LIBRARY_PATH:./coordinator/verifier/lib"
        CHAIN_ID='534353'
        // LOG_DOCKER = 'true'
    }
    stages {
        stage('Build') {
            parallel {
                stage('Build Prerequisite') {
                    steps {
                        sh 'make dev_docker'
                        sh 'make -C bridge mock_abi'
                        sh 'make -C common/bytecode all'
                    }
                }
                stage('Check Bridge Compilation') {
                    steps {
                        sh 'make -C bridge bridge_bins'
                    }
                }
                stage('Check Coordinator Compilation') {
                    steps {
                        sh 'export PATH=/home/ubuntu/go/bin:$PATH'
                        sh 'make -C coordinator coordinator'
                    }
                }
                stage('Check Database Compilation') {
                    steps {
                        sh 'make -C database db_cli'
                    }
                }
                stage('Check Database Docker Build') {
                    steps {
                        sh 'make -C database docker'
                    }
                }
            }
        }
        stage('Parallel Test') {
            parallel{
                stage('Race test common package') {
                    steps {
                        sh 'go test -v -race -coverprofile=coverage.common.txt -covermode=atomic scroll-tech/common/...'
                    }
                }
                stage('Race test bridge package') {
                    steps {
                        sh "cd ./bridge && ../build/run_tests.sh bridge"
                    }
                }
                stage('Race test coordinator package') {
                    steps {
                        sh 'cd ./coordinator && go test -exec "env LD_LIBRARY_PATH=${PWD}/verifier/lib" -v -race -gcflags="-l" -ldflags="-s=false" -coverpkg="scroll-tech/coordinator" -coverprofile=../coverage.coordinator.txt -covermode=atomic ./...'
                    }
                }
                stage('Race test database package') {
                    steps {
                        sh 'go test -v -race -coverprofile=coverage.db.txt -covermode=atomic scroll-tech/database/...'
                    }
                }
                stage('Integration test') {
                    steps {
                        sh 'go test -v -tags="mock_prover mock_verifier" -p 1 scroll-tech/integration-test/...'
                    }
                }
            }
        }
        stage('Compare Coverage') {
            steps {
                sh './build/post-test-report-coverage.sh'
                script {
                    currentBuild.result = 'SUCCESS'
                }
                step([$class: 'CompareCoverageAction', publishResultAs: 'Comment', scmVars: [GIT_URL: env.GIT_URL]])
            }
        }
    }
    post {
        always {
            publishCoverage adapters: [coberturaReportAdapter(path: 'cobertura.xml', thresholds: [[thresholdTarget: 'Aggregated Report', unhealthyThreshold: 40.0]])], checksName: '', sourceFileResolver: sourceFiles('NEVER_STORE') 
            cleanWs() 
            slackSend(message: "${JOB_BASE_NAME} ${GIT_COMMIT} #${BUILD_NUMBER} deploy ${currentBuild.result}")
        }
    }
}
