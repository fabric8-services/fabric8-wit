#!groovy

// Pipeline documentation: https://jenkins.io/doc/pipeline/
// Groovy syntax reference: http://groovy-lang.org/syntax.html

def projectDescription = '''
This project is awesome.
'''

// Node executes on 64bit linux only
//node('unix && 64bit') {
node {
    // no longer needed if node ('linux && 64bit') was used...
    if (!isUnix()) {
        error "This file can only run on unix-like systems."
    }

    stage 'checkout'
    checkout scm

    // Build a custom docker image for building the project
    stage 'create builder image'
    def builderImageTag = "almighty-core-builder-image:" + env.BRANCH_NAME + "-" + env.BUILD_NUMBER
    def builderImageDir = "jenkins/docker/builder"
    def builderImage = docker.build(builderImageTag, builderImageDir)

    stage 'build with container'
    builderImage.withRun {
        sh 'cat /etc/redhat-release'
        sh 'go version'
        sh 'git --version'
        sh 'hg --version'
        sh 'glide --version'

        sh 'make deps'
        sh 'make generate'
        sh 'make build'
        sh 'make test-unit'
    }

    // Can be used when executing downloaded glide tool
    // withEnv(["PATH+MAVEN=${tool 'M3'}/bin"]) {
    //   sh 'mvn -B verify'
    // }
    // or env.PATH = "${nodeHome}/bin:${env.PATH}"

    stage 'output branch name'
    echo env.BRANCH_NAME

    stage 'docker tool'
    // Ensure that the docker tool is installed somewhere accessible to Jenkins
    def dockerTool = tool 'docker'

    sh 'echo hello world'
}

// Don't use "input" within a "node"
// When you use inputs, it is a best practice to wrap them in timeouts. Wrapping inputs in timeouts allows them to be cleaned up if
// approvals do not occur within a given window. For example:
//
// timeout(time:5, unit:'DAYS') {
//     input message:'Approve deployment?', submitter: 'it-ops'
// }

// Try catch blocks:
//
//     try {
//         sh 'might fail'
//         mail subject: 'all well', to: 'admin@somewhere', body: 'All well.'
//     } catch (e) {
//         def w = new StringWriter()
//         e.printStackTrace(new PrintWriter(w))
//         mail subject: "failed with ${e.message}", to: 'admin@somewhere', body: "Failed: ${w}"
//         throw e
//     }

// For headless GUI tests see https://github.com/jenkinsci/workflow-basic-steps-plugin/blob/master/CORE-STEPS.md#build-wrappers
