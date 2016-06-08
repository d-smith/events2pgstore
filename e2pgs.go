package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	consulapi "github.com/hashicorp/consul/api"
	_ "github.com/lib/pq"
	"github.com/xtraclabs/pgeventstore"
	"github.com/xtraclabs/snspublish/db"
	"log"
	"os"
	"strings"
)

const usage = `

e2pgs - events to Postgres store

This software requires configuration via consul or environment variables.
To read configuration from consul specify the consul API endpoint via
the CONSUL_ADDR env variable, and provide the root of the config key path
via the CONSUL_KEYROOT env variable.



When configuring from environmen variables, the following environment
variables are required:

QUEUE_URL - AWS SQS queue to read from
DB_HOST - database host
DB_PORT - database port
DB_NAME - database name
DB_USER - database user name
DB_PASSWORD - databaase password
AWS_REGION - AWS region to use

You might also want to use AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY
when running outside an AWS context.

`

var (
	awsregion  string
	queueUrl   string
	dbname     string
	dbhost     string
	dbuser     string
	dbpassword string
	dbport     string
)

func consulClientKVFromConfig(hostport string) (*consulapi.KV, error) {
	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = hostport
	client, err := consulapi.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}

	return client.KV(), nil
}

func getValForKey(kvClient *consulapi.KV, key string) (string, error) {
	//log.Println("Get key", key)
	kvPair, _, err := kvClient.Get(key, nil)
	if err != nil {
		//	log.Println("error on key get", err.Error())
		return "", err
	}

	if kvPair == nil {
		//	log.Println("nil kvpair")
		return "", nil
	}

	//log.Println("returning key value", string(kvPair.Value))
	return string(kvPair.Value), nil
}

func initFromConsul() (bool, error) {
	consulAddr := os.Getenv("CONSUL_ADDR")
	if consulAddr == "" {
		return false, nil
	}

	consulKeyRoot := os.Getenv("CONSUL_KEY_ROOT")
	if consulKeyRoot == "" {
		return false, errors.New("CONSUL_KEYROOT must also be provided when CONSUL_ADDR is specified.")
	}

	if !strings.HasSuffix(consulKeyRoot, "/") {
		consulKeyRoot = consulKeyRoot + "/"
	}

	kvClient, err := consulClientKVFromConfig(consulAddr)
	if err != nil {
		log.Println("Unable to connect to consul", err.Error())
		return false, err
	}

	if awsregion, err = getValForKey(kvClient, consulKeyRoot+"awsRegion"); err != nil {
		return false, err
	}

	if queueUrl, err = getValForKey(kvClient, consulKeyRoot+"queueUrl"); err != nil {
		return false, err
	}

	if dbhost, err = getValForKey(kvClient, consulKeyRoot+"dbHost"); err != nil {
		return false, err
	}

	if dbname, err = getValForKey(kvClient, consulKeyRoot+"dbName"); err != nil {
		return false, err
	}

	if dbuser, err = getValForKey(kvClient, consulKeyRoot+"dbUser"); err != nil {
		return false, err
	}

	if dbpassword, err = getValForKey(kvClient, consulKeyRoot+"dbPassword"); err != nil {
		return false, err
	}

	if dbport, err = getValForKey(kvClient, consulKeyRoot+"dbPort"); err != nil {
		return false, err
	}

	return true, nil
}

func initFromEnv() {
	queueUrl = os.Getenv("QUEUE_URL")
	dbhost = os.Getenv("DB_HOST")
	dbname = os.Getenv("DB_NAME")
	dbuser = os.Getenv("DB_USER")
	dbpassword = os.Getenv("DB_PASSWORD")
	dbport = os.Getenv("DB_PORT")
	awsregion = os.Getenv("AWS_REGION")
}

func openDB() (*sql.DB, error) {
	connectStr := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable", dbuser, dbpassword, dbname, dbhost, dbport)
	log.Println("open db - ", connectStr)
	db, err := sql.Open("postgres", connectStr)
	if err != nil {
		return nil, err
	}

	err = db.QueryRow("select where 1 = 1").Scan()
	return db, err

}

func isInitializedAsExpected() bool {

	log.Println("Configuration: ")
	log.Println("\tqueueUrl", queueUrl)
	log.Println("\tdbhost", dbhost)
	log.Println("\tdbname", dbname)
	log.Println("\tdbuser", dbuser)
	if dbpassword == "" {
		log.Println("\tdbpassword", "")
	} else {
		log.Println("\t*********")
	}
	log.Println("\tdbport", dbport)
	log.Println("\tawsregion", awsregion)

	if queueUrl == "" || dbhost == "" || dbname == "" || dbuser == "" || dbpassword == "" || dbport == "" {
		return false
	}

	return true
}

func main() {
	consulInit, err := initFromConsul()
	if err != nil {
		log.Println("Error initializing from consul", err.Error())
		log.Println(usage)
		return
	}

	if !consulInit {
		initFromEnv()
	}

	if !isInitializedAsExpected() {
		log.Println(usage)
		log.Println("Not all configuration parameters were provided - exiting.")
		return
	}

	copydb, err := openDB()
	if err != nil {
		log.Fatal(err)
	}

	type message struct {
		Message string
	}

	params := &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueUrl),
		MaxNumberOfMessages: aws.Int64(1),
		VisibilityTimeout:   aws.Int64(1),
		WaitTimeSeconds:     aws.Int64(5),
	}

	//If no value of AWS Region is provided try defaulting to us-east-1.
	if awsregion == "" {
		awsregion = "us-east-1"
	}

	svc := sqs.New(session.New(), &aws.Config{Region: aws.String(awsregion)})
	for {
		log.Println("check messages...")
		resp, err := svc.ReceiveMessage(params)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		for _, msg := range resp.Messages {

			var theMessage message
			err := json.Unmarshal([]byte(*msg.Body), &theMessage)
			log.Println("event message", theMessage.Message)

			aggId, version, payload, typecode, err := db.DecodePGEvent(theMessage.Message)
			if err != nil {
				log.Println("::can't decode message")
			} else {
				err := pgeventstore.InsertEventFromParts(copydb, aggId, version, typecode, payload)
				if err != nil {
					log.Println("::can't store message", err.Error())
				}
			}

			delParams := &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(queueUrl),
				ReceiptHandle: msg.ReceiptHandle,
			}

			_, err = svc.DeleteMessage(delParams)
			if err != nil {
				log.Println("Error deleting response", err.Error())
			}

		}
	}
}
