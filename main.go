package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go/aws"
	awsSess "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func main() {
	dynamodbTable := os.Getenv("DYNAMODB_TABLE")
	dynamodbRecordID := os.Getenv("DYNAMODB_RECORD_ID")
	sourceDB := os.Getenv("SOURCE_DB")
	destDB := os.Getenv("DEST_DB")
	region := os.Getenv("AWS_REGION")

	sess, err := awsSess.NewSession()
	if err != nil {
		log.Printf("cannot create new aws session, error: %v", err)
		return
	}
	dynamodbSvc := dynamodb.New(sess, &aws.Config{
		Region: aws.String(region),
	})

	_, err = dynamodbSvc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(dynamodbTable),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(dynamodbRecordID),
			},
		},
	})
	if err != nil {
		log.Printf("cannot get item: error %v", err)
		return
	}

	log.Printf("Clone DB %v to %v\n", sourceDB, destDB)
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("mysqldump --defaults-file=/mnt/MYSQL_CNF --set-gtid-purged=OFF --column-statistics=0 --routines --no-tablespaces %v | sed -e 's/DEFINER[ ]*=[ ]*[^*]*\\*/\\*/' | mysql --defaults-file=/mnt/MYSQL_CNF %v", sourceDB, destDB))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()
	log.Printf("Finish clone DB %v to %v with out: %v, err: %v\n", sourceDB, destDB, stdout.String(), stderr.String())

	dynamodbSvc.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(dynamodbTable),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(dynamodbRecordID),
			},
		},
	})
}
