#!groovy

// Pipeline documentation: https://jenkins.io/doc/pipeline/
// Groovy syntax reference: http://groovy-lang.org/syntax.html

// Node executes on 64bit linux only
//node('unix && 64bit') {
node {

  //def err = null
  //currentBuild.result = FAILURE

  // try {

    // no longer needed if node ('linux && 64bit') was used...
    if (!isUnix()) {
        error "This file can only run on unix-like systems."
    }

    def PACKAGE_NAME = 'github.com/almighty/almighty-core'
    def checkoutDir = "go/src/${PACKAGE_NAME}"

    stage 'Checkout from SCM'

      print "Will checkout from SCM into ${checkoutDir}"
      //checkout scm
      checkout changelog: false, poll: false, scm: [$class: 'GitSCM', branches: [[name: "*/${env.BRANCH_NAME}"]], doGenerateSubmoduleConfigurations: false, extensions: [[$class: 'WipeWorkspace'], [$class: 'RelativeTargetDirectory', relativeTargetDir: "go/src/${PACKAGE_NAME}"]], submoduleCfg: [], userRemoteConfigs: [[]]]
      //checkout([
      //  $class: 'GitSCM',
      //  // branches: [[
      //  //   name: '*/' + env.BRANCH_NAME
      //  // ]],
      //  extensions: [
      //    [$class: 'LocalBranch', localBranch: env.BRANCH_NAME],
      //    // Delete the contents of the workspace before building,
      //    // ensuring a fully fresh workspace.
      //    [$class: 'WipeWorkspace'],
      //    // Specify a local directory (relative to the workspace root) where
      //    // the Git repository will be checked out.
      //    // If left empty, the workspace root itself will be used.
      //    [$class: 'RelativeTargetDirectory', relativeTargetDir: "${checkoutDir}"]
      //  ]
      //])

    stage 'Create builder image'

      def builderImageTag = "almighty-core-builder-image:" + env.BRANCH_NAME + "-" + env.BUILD_NUMBER
      // Path to where to find the builder's "Dockerfile"
      def builderImageDir = "jenkins/docker/builder"
      def builderImage = docker.build(builderImageTag, builderImageDir)

    stage 'Build with builder container'

      builderImage.withRun {
        // Setup GOPATH
        env.GOPATH = "${env.WORKSPACE}/go"
        def PACKAGE_PATH = "${env.GOPATH}/src/${PACKAGE_NAME}"
        sh "mkdir -pv ${PACKAGE_PATH}"
        sh "mkdir -pv ${env.GOPATH}/bin"
        sh "mkdir -pv ${env.GOPATH}/pkg"

        sh 'cat /etc/redhat-release'
        sh 'go version'
        sh 'git --version'
        sh 'hg --version'
        sh 'glide --version'

        sh 'make deps'
        sh 'make generate'
        sh 'make build'
        sh 'make test-unit'

        // Add stage inside withRun {} and add a cleanup stage?
      }

    currentBuild.result = "SUCCESS"

  //} catch (e) {

  //  def w = new StringWriter()
  //  err.printStackTrace(new PrintWriter(w))

  //  mail body: "project build error: ${err}" ,
  //  from: 'admin@your-jenkins.com',
  //  replyTo: 'noreply@your-jenkins.com',
  //  subject: 'project build failed',
  //  to: 'kkleine@redhat.com'

  //  throw err
  //}
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
