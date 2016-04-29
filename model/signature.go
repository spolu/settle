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

// Signature represents the storage model for a signature.
type Signature struct {
	ID        string
	Created   int64
	Fact      string
	Entity    PublicKey
	Signature string
}

var signatureProjectExpr = "s_id, s_created, s_fact, s_entity, s_signature"
var signatureUpdateExpr = "SET " +
	"s_created = :s_created, " +
	"s_entity = :s_entity, " +
	"s_signature = :s_signature"
var signatureTableName = "signatures"

// NewSignature creates a new signature.
func NewSignature(
	fact string,
	entity PublicKey,
	signature string,
) *Signature {
	return &Signature{
		ID:        token.New("signature", string(entity)),
		Created:   time.Now().UnixNano(),
		Fact:      fact,
		Entity:    entity,
		Signature: signature,
	}
}

// LoadSignature loads a Signature from its ID and the associated Fact ID.
func LoadSignature(
	ID string,
	fact string,
) (*Signature, error) {
	params := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"s_id":   {S: aws.String(ID)},
			"s_fact": {S: aws.String(fact)},
		},
		TableName:            aws.String(signatureTableName),
		ConsistentRead:       aws.Bool(true),
		ProjectionExpression: aws.String(signatureProjectExpr),
	}
	resp, err := svc.GetItem(params)
	if err != nil {
		return nil, errors.Trace(err)
	}

	created, err := strconv.ParseInt(*resp.Item["s_created"].N, 10, 64)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &Signature{
		ID:        ID,
		Created:   created,
		Fact:      *resp.Item["s_fact"].S,
		Entity:    PublicKey(*resp.Item["s_entity"].S),
		Signature: *resp.Item["s_signature"].S,
	}, nil
}

// Save creates or updates the Signature.
func (s *Signature) Save() error {
	params := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"s_id":   {S: aws.String(s.ID)},
			"s_fact": {S: aws.String(s.Fact)},
		},
		TableName: aws.String(signatureTableName),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":s_created":   {N: aws.String(fmt.Sprintf("%d", s.Created))},
			":s_entity":    {S: aws.String(string(s.Entity))},
			":s_signature": {S: aws.String(s.Signature)},
		},
		UpdateExpression: aws.String(signatureUpdateExpr),
	}
	_, err := svc.UpdateItem(params)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}
