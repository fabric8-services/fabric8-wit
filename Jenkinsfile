#!/usr/bin/groovy
@Library('github.com/fabric8io/fabric8-pipeline-library@master')

def utils = new io.fabric8.Utils()

node {

  def envStage = utils.environmentNamespace('staging')
  def envProd = utils.environmentNamespace('production')
  def newVersion = ""

  def PROJECT_NAME = "almighty-core"
  def PACKAGE_NAME = 'github.com/almighty/almighty-core'
  def GOPATH_IN_CONTAINER="/tmp/go"
  def DOCKER_BUILD_DIR = "${env.WORKSPACE}/${PROJECT_NAME}-build"
  def DOCKER_IMAGE_CORE = "${PROJECT_NAME}"
  def DOCKER_IMAGE_DEPLOY = "${PROJECT_NAME}-deploy"
  def DOCKER_RUN_INTERACTIVE_SWITCH = ""
  def BUILD_TAG = "${PROJECT_NAME}-local-build"
  def DOCKER_CONTAINER_NAME = "${BUILD_TAG}"
  def PACKAGE_PATH= "${GOPATH_IN_CONTAINER}/src/${PACKAGE_NAME}"

  clientsNode{

    stage 'Checkout'
    def checkoutDir = "go/src/${PACKAGE_NAME}"
    sh "mkdir -pv ${checkoutDir}"
    dir ("${checkoutDir}") {
      checkout scm
      newVersion = sh(returnStdout: true, script: 'git rev-parse `git rev-parse --abbrev-ref HEAD`').take(6)
    }
    env.setProperty('VERSION',newVersion)


    def CUR_DIR = pwd() + "/${checkoutDir}"

    //def GROUP_ID = sh(returnStdout: true, script: 'id -g').trim()
    //def USER_ID = sh(returnStdout: true, script: 'id -u').trim()
    //echo "-u ${USER_ID}:${GROUP_ID}"

    def namespace = utils.getNamespace()
    def newImageName = "${env.FABRIC8_DOCKER_REGISTRY_SERVICE_HOST}:${env.FABRIC8_DOCKER_REGISTRY_SERVICE_PORT}/${namespace}/${env.JOB_NAME}:${newVersion}"

    container('client') {

      stage 'Create Builder'
      //sh "make docker-start"
      sh "mkdir -p ${DOCKER_BUILD_DIR}"
      sh "docker build -t ${DOCKER_IMAGE_CORE} -f ${CUR_DIR}/Dockerfile.builder ${CUR_DIR}"
      sh "ls -la ${CUR_DIR}"
      sh "docker run --detach=true -t ${DOCKER_RUN_INTERACTIVE_SWITCH} --name=\"${DOCKER_CONTAINER_NAME}\" -e GOPATH=${GOPATH_IN_CONTAINER}	-w ${PACKAGE_PATH} ${DOCKER_IMAGE_CORE}"


      stage 'Get Deps'
      //sh "make docker-deps"
      sh "docker exec -t ${DOCKER_RUN_INTERACTIVE_SWITCH} \"${DOCKER_CONTAINER_NAME}\" bash -ec 'ls -la'"
      sh "docker exec -t ${DOCKER_RUN_INTERACTIVE_SWITCH} \"${DOCKER_CONTAINER_NAME}\" bash -ec 'make deps'"
      
      stage 'Generate'
      //sh "make docker-generate"
      sh "docker exec -t ${DOCKER_RUN_INTERACTIVE_SWITCH} \"${DOCKER_CONTAINER_NAME}\" bash -ec 'make generate'"

      stage 'Compile'
      //sh "make docker-build"
      sh "docker exec -t ${DOCKER_RUN_INTERACTIVE_SWITCH} \"${DOCKER_CONTAINER_NAME}\" bash -ec 'make build'"

      stage 'Run UnitTests'
      //sh "make docker-test-unit"
      sh "docker exec -t ${DOCKER_RUN_INTERACTIVE_SWITCH} \"${DOCKER_CONTAINER_NAME}\" bash -ec 'make test-unit'"

      stage 'Get BuilArifacts'
      sh "docker cp \"${DOCKER_CONTAINER_NAME}\":${PACKAGE_PATH}/bin/ ${CUR_DIR}"

      stage 'Delete Builder'
      sh "docker rm --force ${DOCKER_CONTAINER_NAME}"

      stage 'Create Runtime'
      //sh "make docker-image-deploy"
      sh "docker build -t ${DOCKER_IMAGE_DEPLOY} -f ${CUR_DIR}/Dockerfile.deploy ${CUR_DIR}"

      stage 'Push Runtime'
      sh "docker tag ${DOCKER_IMAGE_DEPLOY} ${newImageName}"
      sh "docker push ${newImageName}"

    }
  }

  def rc = createKubernetesJson {
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

def createKubernetesJson(body) {
    // evaluate the body block, and collect configuration into the object
    def config = [:]
    body.resolveStrategy = Closure.DELEGATE_FIRST
    body.delegate = config
    body()

    def rc = """
    {
      "apiVersion" : "v1",
      "kind" : "Template",
      "labels" : { },
      "metadata" : {
        "annotations" : {
          "description" : "${config.label} example",
          "fabric8.${env.JOB_NAME}/iconUrl" : "${config.icon}"
        },
        "labels" : { },
        "name" : "${env.JOB_NAME}"
      },
      "objects" : [{
        "kind": "Service",
        "apiVersion": "v1",
        "metadata": {
            "name": "${env.JOB_NAME}",
            "creationTimestamp": null,
            "labels": {
                "component": "${env.JOB_NAME}",
                "container": "${config.label}",
                "group": "quickstarts",
                "project": "${env.JOB_NAME}",
                "provider": "fabric8",
                "expose": "true",
                "version": "${config.version}"
            },
            "annotations": {
                "fabric8.io/app-menu": "development",
                "fabric8.io/iconUrl": "https://cdn.rawgit.com/fabric8io/fabric8/master/website/src/images/fabric8_icon.svg",
                "fabric8.${env.JOB_NAME}/iconUrl" : "${config.icon}",
                "prometheus.io/port": "${config.port}",
                "prometheus.io/scheme": "http",
                "prometheus.io/scrape": "true"
            }
        },
        "spec": {
            "ports": [
                {
                    "protocol": "TCP",
                    "port": 80,
                    "targetPort": ${config.port}
                }
            ],
            "selector": {
                "component": "${env.JOB_NAME}",
                "container": "${config.label}",
                "group": "quickstarts",
                "project": "${env.JOB_NAME}",
                "provider": "fabric8",
                "version": "${config.version}"
            },
            "type": "LoadBalancer",
            "sessionAffinity": "None"
        }
    },
    {
        "kind": "ReplicationController",
        "apiVersion": "v1",
        "metadata": {
            "name": "${env.JOB_NAME}",
            "generation": 1,
            "creationTimestamp": null,
            "labels": {
                "component": "${env.JOB_NAME}",
                "container": "${config.label}",
                "group": "quickstarts",
                "project": "${env.JOB_NAME}",
                "provider": "fabric8",
                "expose": "true",
                "version": "${config.version}"
            },
            "annotations": {
                "fabric8.io/iconUrl": "https://cdn.rawgit.com/fabric8io/fabric8/master/website/src/images/fabric8_icon.svg",
                "fabric8.${env.JOB_NAME}/iconUrl" : "${config.icon}"
            }
        },
        "spec": {
            "replicas": 1,
            "selector": {
                "component": "${env.JOB_NAME}",
                "container": "${config.label}",
                "group": "quickstarts",
                "project": "${env.JOB_NAME}",
                "provider": "fabric8",
                "version": "${config.version}"
            },
            "template": {
                "metadata": {
                    "creationTimestamp": null,
                    "labels": {
                        "component": "${env.JOB_NAME}",
                        "container": "${config.label}",
                        "group": "quickstarts",
                        "project": "${env.JOB_NAME}",
                        "provider": "fabric8",
                        "version": "${config.version}"
                    }
                },
                "spec": {
                    "containers": [
                        {
                            "name": "${env.JOB_NAME}",
                            "image": "${env.FABRIC8_DOCKER_REGISTRY_SERVICE_HOST}:${env.FABRIC8_DOCKER_REGISTRY_SERVICE_PORT}/${env.KUBERNETES_NAMESPACE}/${env.JOB_NAME}:${config.version}",
                            "ports": [
                                {
                                    "name": "web",
                                    "containerPort": ${config.port},
                                    "protocol": "TCP"
                                }
                            ],
                            "env": [
                                {
                                    "name": "KUBERNETES_NAMESPACE",
                                    "valueFrom": {
                                        "fieldRef": {
                                            "apiVersion": "v1",
                                            "fieldPath": "metadata.namespace"
                                        }
                                    }
                                },
                                {
                                    "name": "ALMIGHTY_POSTGRES_HOST",
                                    "value": "172.30.209.155"
                                },
                                {
                                    "name": "ALMIGHTY_POSTGRES_PORT",
                                    "value": "5432"
                                },
                                {
                                    "name": "ALMIGHTY_POSTGRES_USER",
                                    "value": "postgres"
                                },
                                {
                                    "name": "ALMIGHTY_POSTGRES_PASSWORD",
                                    "value": "mysecretpassword"
                                }
                            ],
                            "resources": {},
                            "terminationMessagePath": "/dev/termination-log",
                            "imagePullPolicy": "IfNotPresent",
                            "securityContext": {}
                        }
                    ],
                    "restartPolicy": "Always",
                    "terminationGracePeriodSeconds": 30,
                    "dnsPolicy": "ClusterFirst",
                    "securityContext": {}
                }
            }
        },
        "status": {
            "replicas": 0
        }
    }]}
    """

    echo 'using Kubernetes resources:\n' + rc
    return rc

  }