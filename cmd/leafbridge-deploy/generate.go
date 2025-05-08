package main

//go:generate go run -buildvcs=true -tags generate .

// The command above causes "go generate" to run the main() function in
// versioninfogenerator.go. It accomplishes this by specifying the "generate"
// build tag when it invokes "go run". It also requests that version control
// information is included in the build, which is necessary for the version
// info generator to extract the information.

// To prepare file version information for leafbridge-deploy.exe, run
// "go generate" which will produce a leafbridge-deploy.syso file based on
// the most recent commit. The information in the .syso file will
// automatically be incorporated by future "go build" commands.
