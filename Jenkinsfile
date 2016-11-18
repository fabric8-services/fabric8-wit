#!/usr/bin/groovy
@Library('github.com/fabric8io/fabric8-pipeline-library@master')

def utils = new io.fabric8.Utils()

node {

  def envStage = utils.environmentNamespace('staging')
  def envProd = utils.environmentNamespace('production')
  def newVersion = '5'

  def PACKAGE_NAME = 'github.com/almighty/almighty-core'

  clientsNode{

    stage 'Checkout project from SCM'
    def checkoutDir = "go/src/${PACKAGE_NAME}"
    sh "mkdir -pv ${checkoutDir}"
    dir ("${checkoutDir}") {
      checkout scm
    }

    def namespace = utils.getNamespace()
    def newImageName = "${env.FABRIC8_DOCKER_REGISTRY_SERVICE_HOST}:${env.FABRIC8_DOCKER_REGISTRY_SERVICE_PORT}/${namespace}/${env.JOB_NAME}:${newVersion}"

    stage 'Create docker builder image 2'
    sh "make docker-start"
    stage 'Fetch dependencies'
    sh "make docker-deps"
    stage 'Generate structure'
    sh "make docker-generate"
    stage 'Build source'
    sh "make docker-build"
    stage 'Run unit tests'
    sh "make docker-test-unit"
    stage 'Create docker deploy image'
    sh "make docker-image-deploy"

    stage 'Push docker deploy image'
    sh "docker tag almighty-core-deploy ${newImageName}"
    sh "docker push ${newImageName}"
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

  stage 'Approve'
  approve{
    room = null
    version = canaryVersion
    console = fabric8Console
    environment = envStage
  }

  stage 'Rollout Production'
  kubernetesApply(file: rc, environment: envProd)
}
