#!/bin/sh

CFGROOT=demo1
QUEUEURL=https://sqs.us-west-2.amazonaws.com/930295567417/june7
AWSREGION=us-west-2
DBHOST=localhost
DBPORT=5432
DBUSR=escopyusr
DBPASSWD=uh-huh
DBNAME=esdbcopy

curl -X PUT localhost:8500/v1/kv/$CFGROOT/
curl -X PUT localhost:8500/v1/kv/$CFGROOT/queueUrl -d $QUEUEURL
curl -X PUT localhost:8500/v1/kv/$CFGROOT/awsRegion -d $AWSREGION
curl -X PUT localhost:8500/v1/kv/$CFGROOT/dbHost -d $DBHOST
curl -X PUT localhost:8500/v1/kv/$CFGROOT/dbPort -d $DBPORT
curl -X PUT localhost:8500/v1/kv/$CFGROOT/dbUser -d $DBUSR
curl -X PUT localhost:8500/v1/kv/$CFGROOT/dbPassword -d $DBPASSWD
curl -X PUT localhost:8500/v1/kv/$CFGROOT/dbName -d $DBNAME