package model

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/spolu/settl/util/errors"
)

var svc *dynamodb.DynamoDB

func init() {
	svc = dynamodb.New(
		session.New(),
		&aws.Config{
			Region: aws.String("us-west-2"),
		},
	)

	params := &dynamodb.ListTablesInput{}
	resp, err := svc.ListTables(params)
	if err != nil {
		log.Fatal(errors.Trace(err))
	}

	fmt.Println("Initialized models with Tables:")
	for _, t := range resp.TableNames {
		fmt.Println("  " + *t)
	}
}
