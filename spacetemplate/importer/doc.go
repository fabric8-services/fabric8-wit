// Package importer provides functions to help import a space template YAML
// definition into the system. It is separated from the main space template
// package because it pulls in a lot of other packages that might themselves
// need to import the spacetemplate package which would cause an import cycle if
// the importer was in the spacetemplate package.
package importer
