imagePrefix = 'scrolltech'
credentialDocker = 'dockerhub'

pipeline {
    agent any
    options {
        timeout (20)
    }
    tools {
        go 'go-1.18'
        nodejs "nodejs"
    }
    environment {
        GO111MODULE = 'on'
        PATH="/home/ubuntu/.cargo/bin:$PATH"
        LD_LIBRARY_PATH="$LD_LIBRARY_PATH:./coordinator/verifier/lib"
        CHAIN_ID='534353'
        // LOG_DOCKER = 'true'
    }
    stages {
        stage('Build') {
            parallel {
                stage('clean docker containers') {
                    steps {
                        // Clean stopped and exited containers.
                        sh "docker ps -a | grep 'Exited' | awk 'BEGIN {print 0} {print \$1}' | xargs docker stop 2>/dev/null"
                        sh "docker ps -a | grep 'hours ago' | awk 'BEGIN {print 0} {print \$1}' | xargs docker stop 2>/dev/null"
                        sh "docker ps -a | grep 'an hour ago' | awk 'BEGIN {print 0} {print \$1}' | xargs docker stop 2>/dev/null"
                        // Remove all stopped containers
                        sh "docker container prune -f"
                    }
                }
                stage('Build Prerequisite') {
                    steps {
                        sh 'make dev_docker'
                        sh 'make -C bridge mock_abi'
                    }
                }
                stage('Check Bridge Compilation') {
                    steps {
                        sh 'make -C bridge bridge'
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
                stage('Check Bridge Docker Build') {
                    steps {
                        sh 'make -C bridge docker'
                    }
                }
                stage('Check Coordinator Docker Build') {
                    steps {
                        sh 'make -C coordinator docker'
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
                stage('Test bridge package') {
                    steps {
                        sh 'go test -v -coverprofile=coverage.bridge.txt -covermode=atomic -p 1 scroll-tech/bridge/...'
                    }
                }
                stage('Test common package') {
                    steps {
                        sh 'go test -v -coverprofile=coverage.common.txt -covermode=atomic -p 1 scroll-tech/common/...'
                    }
                }
                stage('Test coordinator package') {
                    steps {
                        sh 'go test -v -coverprofile=coverage.coordinator.txt -covermode=atomic -p 1 scroll-tech/coordinator/...'
                    }
                }
                stage('Test database package') {
                    steps {
                        sh 'go test -v -coverprofile=coverage.db.txt -covermode=atomic -p 1 scroll-tech/database/...'
                    }
                }
                stage('Integration test') {
                    steps {
                        sh 'go test -v -tags="mock_prover mock_verifier" -coverprofile=coverage.integration.txt -covermode=atomic -p 1 scroll-tech/integration-test/...'
                    }
                }
                stage('Race test bridge package') {
                    steps {
                        sh 'go test -v -race -coverprofile=coverage.txt -covermode=atomic scroll-tech/bridge/...'
                    }
                }
                stage('Race test coordinator package') {
                    steps {
                        sh 'go test -v -race -coverprofile=coverage.txt -covermode=atomic scroll-tech/coordinator/...'
                    }
                }
                stage('Race test database package') {
                    steps {
                        sh 'go test -v -race -coverprofile=coverage.txt -covermode=atomic scroll-tech/database/...'
                    }
                }
            }
        }
        stage('Compare Coverage') {
            steps {
                sh "./build/post-test-report-coverage.sh"
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
