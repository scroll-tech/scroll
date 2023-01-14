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
        // LOG_DOCKER = 'true'
    }
    stages {
        stage('Build Prerequisite') {
            when {
                anyOf {
                    changeset "Jenkinsfile"
                    changeset "build/**"
                    changeset "go.work**"
                    changeset "bridge/**"
                    changeset "coordinator/**"
                    changeset "common/**"
                    changeset "database/**"
                    changeset "tests/**"
                }
            }
            steps {
                sh 'make dev_docker'
                sh 'make -C bridge mock_abi'
                sh 'export PATH=/home/ubuntu/go/bin:$PATH'
                sh 'make -C coordinator libzkp'
            }
        }
        stage('Parallel Test') {
            when {
                anyOf {
                    changeset "Jenkinsfile"
                    changeset "build/**"
                    changeset "go.work**"
                    changeset "bridge/**"
                    changeset "coordinator/**"
                    changeset "common/**"
                    changeset "database/**"
                    changeset "tests/**"
                }
            }
            parallel{
                // check compilations
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

                // test packages/integration
                stage('Test bridge package') {
                    steps {
                        sh 'go test -v -race -coverprofile=coverage.bridge.txt -covermode=atomic -p 1 scroll-tech/bridge/...'
                    }
                }
                stage('Test common package') {
                    steps {
                        sh 'go test -v -race -coverprofile=coverage.common.txt -covermode=atomic -p 1 scroll-tech/common/...'
                    }
                }
                stage('Test coordinator package') {
                    steps {
                        sh 'go test -v -race -coverprofile=coverage.coordinator.txt -covermode=atomic -p 1 scroll-tech/coordinator/...'
                    }
                }
                stage('Test database package') {
                    steps {
                        sh 'go test -v -race -coverprofile=coverage.db.txt -covermode=atomic -p 1 scroll-tech/database/...'
                    }
                }
                stage('Integration test') {
                    steps {
                        sh 'go test -v -race -tags="mock_prover mock_verifier" -coverprofile=coverage.integration.txt -covermode=atomic -p 1 scroll-tech/integration-test/...'
                    }
                }

                // race test packages
                stage('Race test bridge package') {
                    steps {
                        sh "cd bridge && go test -v -race -coverprofile=coverage.txt -covermode=atomic \$(go list ./... | grep -v 'database\\|common\\|l1\\|l2\\|coordinator')"
                    }
                }
                stage('Race test coordinator package') {
                    steps {
                        sh "cd coordinator && go test -v -race -coverprofile=coverage.txt -covermode=atomic \$(go list ./... | grep -v 'database\\|common\\|l1\\|l2\\|coordinator')"
                    }
                }
                stage('Race test database package') {
                    steps {
                        sh "cd database && go test -v -race -coverprofile=coverage.txt -covermode=atomic \$(go list ./... | grep -v 'database\\|common\\|l1\\|l2\\|coordinator')"
                    }
                }
            }
        }
        stage('Compare Coverage') {
            when {
                anyOf {
                    changeset "Jenkinsfile"
                    changeset "build/**"
                    changeset "go.work**"
                    changeset "bridge/**"
                    changeset "coordinator/**"
                    changeset "common/**"
                    changeset "database/**"
                    changeset "tests/**"
                }
            }
            steps {
                sh "./build/post-test-report-coverage.sh"
                script {
                    currentBuild.result = 'SUCCESS'
                }
                step([$class: 'CompareCoverageAction', publishResultAs: 'statusCheck', scmVars: [GIT_URL: env.GIT_URL]])
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
