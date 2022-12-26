imagePrefix = 'scrolltech'
credentialDocker = 'dockerhub'

pipeline {
    agent any
    options {
        timeout (20)
    }
    tools {
        go 'go-1.18'
    }
    environment {
        GO111MODULE = 'on'
        PATH="/home/ubuntu/.cargo/bin:$PATH"
        // LOG_DOCKER = 'true'
    }
    stages {
        stage('Build') {
            when {
                anyOf {
                    changeset "Jenkinsfile"
                    changeset "build/**"
                    changeset "go.work**"
                    changeset "bridge/**"
                    changeset "coordinator/**"
                    changeset "common/**"
                    changeset "database/**"
                }
            }
            parallel {
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
        stage('Test') {
            when {
                anyOf {
                    changeset "Jenkinsfile"
                    changeset "build/**"
                    changeset "go.work**"
                    changeset "bridge/**"
                    changeset "coordinator/**"
                    changeset "common/**"
                    changeset "database/**"
                }
            }
            steps {
               catchError(buildResult: 'FAILURE', stageResult: 'FAILURE') {
                    sh '''
                        go test -v -race -coverprofile=coverage.db.txt -covermode=atomic -p 1 scroll-tech/database/...
                        go test -v -race -coverprofile=coverage.bridge.txt -covermode=atomic -p 1 scroll-tech/bridge/...
                        go test -v -race -coverprofile=coverage.common.txt -covermode=atomic -p 1 scroll-tech/common/...
                        go test -v -race -coverprofile=coverage.coordinator.txt -covermode=atomic -p 1 scroll-tech/coordinator/...
                        sh build/post-test-report-coverage.sh
                    '''
                    script {
                        for (i in ['bridge', 'coordinator', 'database']) {
                            sh "cd $i && go test -v -race -coverprofile=coverage.txt -covermode=atomic \$(go list ./... | grep -v 'database\\|l2\\|l1\\|common\\|coordinator')"
                        }
                    }
               }
            }
        }
        stage('Coverage') {
            steps { 
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
