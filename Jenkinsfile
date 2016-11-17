#!/usr/bin/groovy
@Library('github.com/fabric8io/fabric8-pipeline-library@master')

def utils = new io.fabric8.Utils()


// Pipeline documentation: https://jenkins.io/doc/pipeline/
// Groovy syntax reference: http://groovy-lang.org/syntax.html

// Only keep the 10 most recent builds
properties([
  [
    $class: 'BuildDiscarderProperty',
      strategy: [
      $class: 'LogRotator',
        numToKeepStr: '10',
        artifactNumToKeepStr: '10',
      ]
  ]
])

try {
  def PACKAGE_NAME = 'github.com/almighty/almighty-core'

  def envStage = utils.environmentNamespace('staging')
  def newVersion = ''

  clientsNode{

    stage 'Checkout project from SCM'
    def checkoutDir = "go/src/${PACKAGE_NAME}"
    sh "mkdir -pv ${checkoutDir}"
    dir ("${checkoutDir}") {
      checkout scm
    }

    // Determine git revision ID
    //sh 'git rev-parse HEAD > GIT_COMMIT'
    //shortCommit = readFile('GIT_COMMIT').take(6)
    shortCommit = "abcd"

    stage 'Create docker builder image'
    def builderImageTag = "almighty-core-builder-image:${env.BRANCH_NAME}-${shortCommit}-${env.BUILD_NUMBER}"
    // Path to where to find the builder's "Dockerfile"
    def builderImageDir = "jenkins/docker/builder"
    def builderImage = docker.build(builderImageTag, builderImageDir)

    builderImage.withRun {c ->
      // Setup GOPATH
      def currentDir = pwd()
      def GOPATH = "${currentDir}/go"
      def PACKAGE_PATH = "${GOPATH}/src/${PACKAGE_NAME}"

      dir ("${PACKAGE_PATH}") {
        env.GOPATH = "${GOPATH}"
        stage "Fetch Go package dependencies"
        sh 'make deps'
        stage "Generate controllers from Goa design code"
        sh 'make generate'
        stage "Go build"
        sh 'make build'
        stage "Run unit tests"
        sh 'make test-unit'
        //stage "Run integration tests"
        //sh 'make test-integration'
        // TODO: (kwk) a cleanup stage?
      }

      newVersion = performCanaryRelease {}

      //stage "Archive artifacts"
      //step([$class: 'ArtifactArchiver',
      //  artifacts: 'alm*',
      //  fingerprint: true])

      // sh "docker logs ${c.id}"
    }
  } // end of node {}

  def rc = getKubernetesJson {
    port = 8080
    label = 'golang'
    icon = 'https://cdn.rawgit.com/fabric8io/fabric8/dc05040/website/src/images/logos/gopher.png'
    version = newVersion
    imageName = clusterImageName
  }


} catch (exc) {
  echo "An error occured. Handling it now."

  //def w = new StringWriter()
  //exc.printStackTrace(new PrintWriter(w))

  emailext subject: "${env.JOB_NAME} (${env.BUILD_NUMBER}) failed",
    body: "It appears that ${env.BUILD_URL} is failing, somebody should do something about that",
    to: 'kkleine@redhat.com',
    recipientProviders: [
      // Sends email to all the people who caused a change in the change set:
      [$class: 'DevelopersRecipientProvider'],
      // Sends email to the user who initiated the build:
      [$class: 'RequesterRecipientProvider']
    ],
    replyTo: 'noreply@localhost',
    attachLog: true

  throw err
}

/*
node {
  def envStage = utils.environmentNamespace('staging')
  def newVersion = ''

  clientsNode{
    git GIT_URL

    stage 'Canary release'
    echo 'NOTE: running pipelines for the first time will take longer as build and base docker images are pulled onto the node'
    if (!fileExists ('Dockerfile')) {
      writeFile file: 'Dockerfile', text: 'FROM golang:onbuild'
    }

    newVersion = performCanaryRelease {}
  }

  def rc = getKubernetesJson {
    port = 8080
    label = 'golang'
    icon = 'https://cdn.rawgit.com/fabric8io/fabric8/dc05040/website/src/images/logos/gopher.png'
    version = newVersion
    imageName = clusterImageName
  }

  stage 'Rollout Staging'
  kubernetesApply(file: rc, environment: envStage)

}

*/