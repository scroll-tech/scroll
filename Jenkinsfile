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
            steps { 
                // start to build project
                catchError(buildResult: 'FAILURE', stageResult: 'FAILURE') {
                    sh '''#!/bin/bash
                        export PATH=/home/ubuntu/go/bin:$PATH
                        make dev_docker
                        make -C bridge mock_abi
                        # check compilation
                        make -C bridge bridge
                        make -C coordinator coordinator
                        make -C database db_cli
                        # check docker build
                        make -C bridge docker
                        make -C coordinator docker
                        make -C database docker
                        '''
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
               sh "docker ps -aq | xargs -r docker stop"
               sh "docker container prune -f"
               catchError(buildResult: 'FAILURE', stageResult: 'FAILURE') {
                    sh '''
                        go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 scroll-tech/database/...
                        go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 scroll-tech/bridge/...
                        go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 scroll-tech/common/...
                        go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 scroll-tech/coordinator/...
                        cd ..
                    '''
                    script {
                        for (i in ['bridge', 'coordinator', 'database']) {
                            sh "cd $i && go test -v -race -coverprofile=coverage.txt -covermode=atomic \$(go list ./... | grep -v 'database\\|l2\\|l1\\|common\\|coordinator')"
                        }
                    }
               }
            }
        }
    }
    post { 
        always { 
            cleanWs() 
            slackSend(message: "${JOB_BASE_NAME} ${GIT_COMMIT} #${BUILD_NUMBER} deploy ${currentBuild.result}")
        }
    }
}
