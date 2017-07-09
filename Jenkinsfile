#!/usr/bin/groovy
@Library('github.com/hectorj2f/fabric8-pipeline-library@wit_pipeline')
def dummy
goTemplate{
  dockerNode{
    if (env.BRANCH_NAME.startsWith('PR-')) {
      goCI{
        githubOrganisation = 'fabric8-services'
        dockerOrganisation = 'fabric8'
        project = 'fabric8-wit'
        dockerBuildOptions = '--file Dockerfile.deploy'
        dockerfileBuilder = '--file Dockerfile.builder'
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
}
