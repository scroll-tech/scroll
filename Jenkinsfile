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
        stage('Clean up environment for testing') {
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
                sh "docker ps -aq | xargs -r docker stop"
                sh "docker container prune -f"
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
            // catchError(buildResult: 'FAILURE', stageResult: 'FAILURE') {
                parallel{
                    stage('Test bridge package') {
                        steps {
                            sh 'go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 scroll-tech/bridge/...'
                        }
                    }
                    stage('Test common package') {
                        steps {
                            sh 'go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 scroll-tech/common/...'
                        }
                    }
                    stage('Test coordinator package') {
                        steps {
                            sh 'go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 scroll-tech/coordinator/...'
                        }
                    }
                    stage('Test database package') {
                        steps {
                            sh 'go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 scroll-tech/database/...'
                        }
                    }
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
           // }
        }
    }
    post { 
        always { 
            cleanWs() 
            slackSend(message: "${JOB_BASE_NAME} ${GIT_COMMIT} #${BUILD_NUMBER} deploy ${currentBuild.result}")
        }
    }
}
