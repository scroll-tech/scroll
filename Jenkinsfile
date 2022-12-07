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
               sh "docker ps -aq | xargs -r docker stop"
               sh "docker container prune -f"
               sh "go install github.com/axw/gocov/gocov@latest"
               sh "go install github.com/AlekSi/gocov-xml@latest"
               catchError(buildResult: 'FAILURE', stageResult: 'FAILURE') {
                    sh '''
                        gocov -v -race -covermode=atomic -p 1 scroll-tech/database/... > coverage.db.json
                        gocov -v -race -covermode=atomic -p 1 scroll-tech/bridge/... > coverage.bridge.json
                        gocov -v -race -covermode=atomic -p 1 scroll-tech/common/... > coverage.common.json
                        gocov -v -race -covermode=atomic -p 1 scroll-tech/coordinator/... > coverage.coordinator.json
                        jq -n '{ Packages: [ inputs.Packages ] | add }' coverage.bridge.json coverage.db.json coverage.coordinator.json coverage.common.json | gocov-xml > coverage.xml
                        cd ..
                    '''
                    script { test_result = true }
               }
            }
        }
        stage("PR Coverage to Github") {
            when { allOf {not { branch 'staging' }; expression { return env.CHANGE_ID != null }} }
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
            cleanWs() 
            slackSend(message: "${JOB_BASE_NAME} ${GIT_COMMIT} #${BUILD_NUMBER} deploy ${currentBuild.result}")
        }
    }
}
