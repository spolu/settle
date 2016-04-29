package model

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/spolu/settl/util/errors"
	"github.com/spolu/settl/util/token"
)

// Revocation represents the storage model for a revocation.
type Revocation struct {
	ID        string
	Created   int64
	Fact      string
	Entity    PublicKey
	Signature string
}

var revocationProjectExpr = "s_id, s_created, s_fact, s_entity, s_signature"
var revocationUpdateExpr = "SET " +
	"s_created = :s_created, " +
	"s_entity = :s_entity, " +
	"s_signature = :s_signature"
var revocationTableName = "revocations"

// NewRevocation creates a new revocation.
func NewRevocation(
	fact string,
	entity PublicKey,
	signature string,
) *Revocation {
	return &Revocation{
		ID:        token.New("revocation", string(entity)),
		Created:   time.Now().UnixNano(),
		Fact:      fact,
		Entity:    entity,
		Signature: signature,
	}
}

// LoadRevocation loads a Revocation from its ID and the associated Fact ID.
func LoadRevocation(
	ID string,
	fact string,
) (*Revocation, error) {
	params := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"s_id":   {S: aws.String(ID)},
			"s_fact": {S: aws.String(fact)},
		},
		TableName:            aws.String(revocationTableName),
		ConsistentRead:       aws.Bool(true),
		ProjectionExpression: aws.String(revocationProjectExpr),
	}
	resp, err := svc.GetItem(params)
	if err != nil {
		return nil, errors.Trace(err)
	}

	created, err := strconv.ParseInt(*resp.Item["s_created"].N, 10, 64)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &Revocation{
		ID:        ID,
		Created:   created,
		Fact:      *resp.Item["s_fact"].S,
		Entity:    PublicKey(*resp.Item["s_entity"].S),
		Signature: *resp.Item["s_signature"].S,
	}, nil
}

// Save creates or updates the Revocation.
func (r *Revocation) Save() error {
	params := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"s_id":   {S: aws.String(r.ID)},
			"s_fact": {S: aws.String(r.Fact)},
		},
		TableName: aws.String(revocationTableName),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":s_created":   {N: aws.String(fmt.Sprintf("%d", r.Created))},
			":s_entity":    {S: aws.String(string(r.Entity))},
			":s_signature": {S: aws.String(r.Signature)},
		},
		UpdateExpression: aws.String(revocationUpdateExpr),
	}
	_, err := svc.UpdateItem(params)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}
