pipeline {
  agent none
  options {
    buildDiscarder logRotator(artifactDaysToKeepStr: '5', artifactNumToKeepStr: '5', daysToKeepStr: '5', numToKeepStr: '5')
    durabilityHint 'PERFORMANCE_OPTIMIZED'
    timeout(5)
  }
  libraries {
    lib('jpl-core@master') // https://github.com/joostvdg/jpl-core
    // lib('jpl-maven@master') // https://github.com/joostvdg/jpl-maven
  }
  environment {
    COMMIT_INFO = ''
    GIT_REPO    = ''
    GIT_SHA     = ''
    REPO        = 'harbor.10.220.7.70.nip.io/test'
    IMAGE       = 'go-demo'
    TAG         = ''
    TAG_BASE    = "0.1"
    CA_PEM      = """-----BEGIN CERTIFICATE-----
MIID7jCCAtagAwIBAgIURv5DzXSDklERFu4gL2sQBNeRg+owDQYJKoZIhvcNAQEL
BQAwgY4xCzAJBgNVBAYTAk5MMRgwFgYDVQQIEw9UaGUgTmV0aGVybGFuZHMxEDAO
BgNVBAcTB1V0cmVjaHQxFTATBgNVBAoTDEtlYXJvcyBUYW56dTEdMBsGA1UECxMU
S2Vhcm9zIFRhbnp1IFJvb3QgQ0ExHTAbBgNVBAMTFEtlYXJvcyBUYW56dSBSb290
IENBMB4XDTIyMDMyMzE1MzUwMFoXDTI3MDMyMjE1MzUwMFowgY4xCzAJBgNVBAYT
Ak5MMRgwFgYDVQQIEw9UaGUgTmV0aGVybGFuZHMxEDAOBgNVBAcTB1V0cmVjaHQx
FTATBgNVBAoTDEtlYXJvcyBUYW56dTEdMBsGA1UECxMUS2Vhcm9zIFRhbnp1IFJv
b3QgQ0ExHTAbBgNVBAMTFEtlYXJvcyBUYW56dSBSb290IENBMIIBIjANBgkqhkiG
9w0BAQEFAAOCAQ8AMIIBCgKCAQEAyZXDL9W2vu365m//E/w8n1M189a5mI9HcTYa
0xZhnup58Zp72PsgzujI/fQe43JEeC+aIOcmsoDaQ/uqRi8p8phU5/poxKCbe9SM
f1OflLD9k2dwte6OV5kcSUbVOgScKL1wGEo5mdOiTFrEp5aLBUcbUeJMYz2IqLVa
v52H0vTzGfmrfSm/PQb+5qnCE5D88DREqKtWdWW2bCW0HhxVHk6XX/FKD2Z0FHWI
ChejeaiarXqWBI94BANbOAOmlhjjyJekT5hL1gh7BuCLbiE+A53kWnXO6Xb/eyuJ
obr+uHLJldoJq7SFyvxrDd/8LAJD4XMCEz+3gWjYDXMH7GfPWwIDAQABo0IwQDAO
BgNVHQ8BAf8EBAMCAQYwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUfGU50Pe9
YTv5SFvGVOz6R7ddPcUwDQYJKoZIhvcNAQELBQADggEBAHMoNDxy9/kL4nW0Bhc5
Gn0mD8xqt+qpLGgChlsMPNR0xPW04YDotm+GmZHZg1t6vE8WPKsktcuv76d+hX4A
uhXXGS9D0FeC6I6j6dOIW7Sbd3iAQQopwICYFL9EFA+QAINeY/Y99Lf3B11JfLU8
jN9uGHKFI0FVwHX428ObVrDi3+OCNewQ3fLmrRQe6F6q2OU899huCg+eYECWvxZR
a3SlVZmYnefbA87jI2FRHUPqxp4P2mDwj/RZxhgIobhw0zz08sqC6DW0Aj1OIJe5
sDAm0uiUdqs7FZN2uKkLKekdTgW0QkTFEJTk5Yk9t/hOrjnHoWQfB+mLhO3vPhip
vhs=
-----END CERTIFICATE-----
"""
  }
  stages {
    stage('Image Build') {
      parallel {
        stage('Kaniko') {
          agent {
            kubernetes {
            label 'kaniko-jre-test'
            yaml """
kind: Pod
spec:
  containers:
  - name: kaniko
    image: gcr.io/kaniko-project/executor:debug
    imagePullPolicy: Always
    command:
    - sleep
    args:
    - 9999999
    volumeMounts:
      - name: jenkins-docker-cfg
        mountPath: /kaniko/.docker
    env:
      - name: SSL_CERT_FILE
        value: /home/jenkins/agent/workspace/kaniko-test-2/ca.pem # this is the job name...
  volumes:
  - name: jenkins-docker-cfg
    projected:
      sources:
      - secret:
          name: harbor-registry-creds
          items:
            - key: .dockerconfigjson
              path: config.json
"""
            }
          }
          stages {
            stage('Checkout') {
              steps {
                script {
                  // use this if used within Multibranch or Org Job
                  scmVars = checkout scm
                  GIT_SHA = "${scmVars.GIT_COMMIT}"
                  COMMIT_INFO = "${scmVars.GIT_COMMIT} ${scmVars.GIT_PREVIOUS_COMMIT}"
                  def changeSetData = sh returnStdout: true, script: "git diff-tree --no-commit-id --name-only -r ${COMMIT_INFO}"
                  // use this if used within a Pipeline Job
                  // scmVars = git('https://github.com/joostvdg/go-demo.git')
                }
                echo "scmVars=${scmVars}"
                echo "changeSetData=${changeSetData}"
                gitRemoteConfigByUrl(scmVars.GIT_URL, 'githubtoken')
                sh '''
                git config --global user.email "jenkins@jenkins.io"
                git config --global user.name "Jenkins"
                '''
                //sh 'env'
              }
            }
            stage('Version Bump') {
              // disable when {} when used in a Pipeline
              when { branch 'main' }
              // requires: https://plugins.jenkins.io/pipeline-utility-steps
              environment {
                TAG = gitNextSemverTag("${TAG_BASE}")
              }
              steps {
                gitTag("v${TAG}")
              }
            }
            stage('Build with Kaniko') {
              when { branch 'main' }
              steps {
                writeFile encoding: 'UTF-8', file: 'ca.pem', text: "${CA_PEM}"
                sh 'echo image fqn=${REPO}/${IMAGE}:${TAG}'
                container(name: 'kaniko', shell: '/busybox/sh') {
                  withEnv(['PATH+EXTRA=/busybox']) {
                    sh '''#!/busybox/sh
                    /kaniko/executor --context `pwd` --destination ${REPO}/${IMAGE}:${TAG} --destination ${REPO}/${IMAGE}:latest --reproducible --label org.opencontainers.image.revision=$GIT_SHA --label org.opencontainers.image.source=$GIT_REPO
                    '''
                  }
                }
              }
            }
          }
        }
      }
    }
    stage('Image Test') {
      when { changeRequest target: 'main' }
      parallel {
        stage('Application Image') {
            agent {
            kubernetes {
              label 'app-test'
              containerTemplate {
                name 'app'
                image "${REPO}/${IMAGE}:${TAG}"
                ttyEnabled true
                command 'cat'
              }
            }
          }
          stages {
            stage('Verify Image') {
              steps {
                container('app') {
                  sh 'echo "hello"'
                }
              }
            }
          }
        }
      }
    }
  }
}
