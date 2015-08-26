#!/bin/bash -e
export GOPATH=~/
export PATH=$GOPATH/bin:/usr/local/go/bin:$PATH
WORKSPACE=$GOPATH/src/github.com/dropbox/changes-client
cd $WORKSPACE
go get -v github.com/jstemmer/go-junit-report
sudo PATH=$PATH GOPATH=$GOPATH `which go` test ./... -timeout=120s -v -race | tee test.log
echo Generating junit.xml...
go-junit-report -set-exit-code < test.log > junit.xml
