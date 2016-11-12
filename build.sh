#!/usr/bin/env bash

echo "build commons"
go build -v github.com/coffeehc/commons
echo "build commons/httpcommons/client"
go build -v github.com/coffeehc/commons/httpcommons/client
