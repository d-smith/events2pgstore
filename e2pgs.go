package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	_ "github.com/lib/pq"
	"github.com/xtraclabs/pgeventstore"
	"github.com/xtraclabs/snspublish/db"
	"log"
	"os"
)

const usage = `
The following environment variables are required:
QUEUE_URL - AWS SQS queue to read from
DB_HOST - database host
DB_PORT - database port
DB_NAME - database name
DB_USER - database user name
DB_PASSWORD - databaase password

You might also want to use AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, and AWS_DEFAULT_REGION
when running outside an AWS context
`

var (
	queueUrl   string
	dbname     string
	dbhost     string
	dbuser     string
	dbpassword string
	dbport     string
)

func initFromEnv() {
	queueUrl = os.Getenv("QUEUE_URL")
	dbhost = os.Getenv("DB_HOST")
	dbname = os.Getenv("DB_NAME")
	dbuser = os.Getenv("DB_USER")
	dbpassword = os.Getenv("DB_PASSWORD")
	dbport = os.Getenv("DB_PORT")

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
	if queueUrl == "" || dbhost == "" || dbname == "" || dbuser == "" || dbpassword == "" || dbport == "" {
		return false
	}

	return true
}

func main() {
	initFromEnv()
	if !isInitializedAsExpected() {
		log.Println(usage)
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

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	svc := sqs.New(session.New(), &aws.Config{Region: aws.String(region)})
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
