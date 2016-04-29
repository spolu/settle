package model

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// FactModel represents the storage model for a fact.
type FactModel struct {
	ID      string
	Created int64
	Entity  Entity
	Type    FctType
	Value   string
}

// LoadFact loads a fact from its ID.
func LoadFact(
	ID string,
) (*FactModel, error) {
	params := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(ID),
			},
		},
		TableName:            aws.String("facts"),
		ConsistentRead:       aws.Bool(true),
		ProjectionExpression: aws.String("id,created,entity,type,value"),
	}
	resp, err := svc.GetItem(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func (f *FactModel) Save() {
	params := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(f.ID),
			},
		},
		TableName:        aws.String("facts"),
		UpdateExpression: aws.String("id,created,entity,type,value"),
	}
	resp, err := svc.UpdateItem(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}
