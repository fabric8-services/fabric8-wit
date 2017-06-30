#!/usr/bin/groovy
@Library('github.com/fabric8io/fabric8-pipeline-library@master')
def dummy
goTemplate{
  dockerNode{
    if (env.BRANCH_NAME.startsWith('PR-')) {
      goCI{
        githubOrganisation = 'fabric8-services'
        dockerOrganisation = 'fabric8'
        project = 'fabric8-wit'
        dockerBuildOptions = '--file Dockerfile.deploy'
      }
    } else if (env.BRANCH_NAME.equals('master')) {
      goRelease{
        githubOrganisation = 'fabric8-services'
        dockerOrganisation = 'fabric8'
        project = 'fabric8-wit'
        dockerBuildOptions = '--file Dockerfile.deploy'
      }
    }
  }
