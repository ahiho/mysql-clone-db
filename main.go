package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

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

	item, err := dynamodbSvc.GetItem(&dynamodb.GetItemInput{
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

	if item.Item == nil {
		log.Println("no flag reset db, do nothing")
		return
	}

	log.Printf("Drop all tables of %v\n", destDB)
	err = executeInShell("echo \"SET FOREIGN_KEY_CHECKS = 0;\" > /tmp/temp.sql")
	if err != nil {
		log.Println(err)
		return
	}

	err = executeInShell(fmt.Sprintf("mysqldump --defaults-file=/mnt/MYSQL_CNF --add-drop-table --set-gtid-purged=OFF --no-data %v | grep 'DROP TABLE' >> /tmp/temp.sql", destDB))
	if err != nil {
		log.Println(err)
		if !strings.HasPrefix(err.Error(), "Warning") {
			log.Println("EXIT")
			return
		}
	}

	err = executeInShell("echo \"SET FOREIGN_KEY_CHECKS = 1;\" >> /tmp/temp.sql")
	if err != nil {
		log.Println(err)
		return
	}

	err = executeInShell(fmt.Sprintf("mysqldump --defaults-file=/mnt/MYSQL_CNF %v < /tmp/temp.sql", destDB))
	if err != nil {
		log.Println(err)
		if !strings.HasPrefix(err.Error(), "Warning") {
			log.Println("EXIT")
			return
		}
	}

	log.Println("Begin clone DB")
	err = executeInShell(fmt.Sprintf("mysqldump --defaults-file=/mnt/MYSQL_CNF --set-gtid-purged=OFF --column-statistics=0 --routines --no-tablespaces %v | sed -e 's/DEFINER[ ]*=[ ]*[^*]*\\*/\\*/' | mysql --defaults-file=/mnt/MYSQL_CNF %v", sourceDB, destDB))
	if err != nil {
		log.Printf("Clone DB %v to %v has err: %v\n", sourceDB, destDB, err.Error())
	} else {
		log.Printf("Finish clone DB %v to %v", sourceDB, destDB)
	}

	dynamodbSvc.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(dynamodbTable),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(dynamodbRecordID),
			},
		},
	})
}

func executeInShell(c string) error {
	cmd := exec.Command("/bin/sh", "-c", c)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return err
	}
	errMsg := stderr.String()
	if errMsg != "" {
		return errors.New(errMsg)
	}
	return nil
}
