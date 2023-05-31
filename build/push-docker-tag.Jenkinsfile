imagePrefix = 'scrolltech'
credentialDocker = 'dockerhub'
TAGNAME = ''
pipeline {
    agent any
    options {
        timeout (20)
    }
    tools {
        go 'go-1.19'
        nodejs "nodejs"
    }
    environment {
        GO111MODULE = 'on'
        PATH="/home/ubuntu/.cargo/bin:$PATH"
        // LOG_DOCKER = 'true'
    }
    stages {
        stage('Tag') {
            steps {
                script {
                    TAGNAME = sh(returnStdout: true, script: 'git tag -l --points-at HEAD')
                    sh "echo ${TAGNAME}"
                    // ... 
                }
            }
        }
        stage('Build') {
            environment {
                // Extract the username and password of our credentials into "DOCKER_CREDENTIALS_USR" and "DOCKER_CREDENTIALS_PSW".
                // (NOTE 1: DOCKER_CREDENTIALS will be set to "your_username:your_password".)
                // The new variables will always be YOUR_VARIABLE_NAME + _USR and _PSW.
                // (NOTE 2: You can't print credentials in the pipeline for security reasons.)
                DOCKER_CREDENTIALS = credentials('dockerhub')
            }
           steps {
               withCredentials([usernamePassword(credentialsId: "${credentialDocker}", passwordVariable: 'dockerPassword', usernameVariable: 'dockerUser')]) {
                    // Use a scripted pipeline.
                    script {
                            stage('Push image') {
                                if (TAGNAME == ""){
                                    return;
                                }
                                sh "docker login --username=$dockerUser --password=$dockerPassword"
                                catchError(buildResult: 'SUCCESS', stageResult: 'SUCCESS') {
                                    script {
                                        try {
                                            sh "docker manifest inspect scrolltech/bridge:$TAGNAME > /dev/null"
                                        } catch (e) {
                                            // only build if the tag non existed
                                            //sh "docker login --username=${dockerUser} --password=${dockerPassword}"
                                            sh "make -C bridge docker"
                                            sh "docker tag scrolltech/bridge:latest scrolltech/bridge:${TAGNAME}"
                                            sh "docker push scrolltech/bridge:${TAGNAME}"
                                            throw e
                                        }
                                    }
                                }
                                catchError(buildResult: 'SUCCESS', stageResult: 'SUCCESS') {
                                    script {
                                        try {
                                            sh "docker manifest inspect scrolltech/coordinator:$TAGNAME > /dev/null"
                                        } catch (e) {
                                            // only build if the tag non existed
                                            //sh "docker login --username=${dockerUser} --password=${dockerPassword}"
                                            sh "make -C coordinator docker"
                                            sh "docker tag scrolltech/coordinator:latest scrolltech/coordinator:${TAGNAME}"
                                            sh "docker push scrolltech/coordinator:${TAGNAME}"
                                            throw e
                                        }
                                    }
                                }
                            }                              
                    }
                }
            }
        }
    }
    post {
        always {
            cleanWs() 
            slackSend(message: "${JOB_BASE_NAME} ${GIT_COMMIT} #${TAGNAME} Tag build ${currentBuild.result}")
        }
    }
}
