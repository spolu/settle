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

// Assertion represents the storage model for an assertion.
type Assertion struct {
	ID        string
	Created   int64
	Fact      string
	Account   PublicKey
	Signature PublicKeySignature
}

var assertionProjectExpr = "s_id, s_created, s_fact, s_account, s_signature"
var assertionUpdateExpr = "SET " +
	"s_created = :s_created, " +
	"s_account = :s_account, " +
	"s_signature = :s_signature"
var assertionTableName = "assertions"

// NewAssertion creates a new assertion.
func NewAssertion(
	fact string,
	account PublicKey,
	signature PublicKeySignature,
) *Assertion {
	return &Assertion{
		ID:        token.New("assertion", string(account)),
		Created:   time.Now().UnixNano(),
		Fact:      fact,
		Account:   account,
		Signature: signature,
	}
}

// LoadAssertion loads a Assertion from its ID and the associated Fact ID.
func LoadAssertion(
	ID string,
	fact string,
) (*Assertion, error) {
	params := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"s_id":   {S: aws.String(ID)},
			"s_fact": {S: aws.String(fact)},
		},
		TableName:            aws.String(assertionTableName),
		ConsistentRead:       aws.Bool(true),
		ProjectionExpression: aws.String(assertionProjectExpr),
	}
	resp, err := svc.GetItem(params)
	if err != nil {
		return nil, errors.Trace(err)
	}

	created, err := strconv.ParseInt(*resp.Item["s_created"].N, 10, 64)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &Assertion{
		ID:        ID,
		Created:   created,
		Fact:      *resp.Item["s_fact"].S,
		Account:   PublicKey(*resp.Item["s_account"].S),
		Signature: PublicKeySignature(*resp.Item["s_signature"].S),
	}, nil
}

// Save creates or updates the Assertion.
func (s *Assertion) Save() error {
	params := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"s_id":   {S: aws.String(s.ID)},
			"s_fact": {S: aws.String(s.Fact)},
		},
		TableName: aws.String(assertionTableName),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":s_created":   {N: aws.String(fmt.Sprintf("%d", s.Created))},
			":s_account":   {S: aws.String(string(s.Account))},
			":s_signature": {S: aws.String(string(s.Signature))},
		},
		UpdateExpression: aws.String(assertionUpdateExpr),
	}
	_, err := svc.UpdateItem(params)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// Verify verifies (in memory) that the assertion corresponds to the fact
// passed as argument and is properly signed.
func (s *Assertion) Verify(
	fact *Fact,
) bool {
	return false
}
