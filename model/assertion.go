package model

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/spolu/settl/util/errors"
	"github.com/spolu/settl/util/token"
	"github.com/stellar/go-stellar-base/keypair"
)

// Assertion represents the storage model for an assertion.
type Assertion struct {
	ID        string
	Created   int64
	Fact      string
	Account   PublicKey
	Signature PublicKeySignature
}

const (
	assertionTableName   string = "assertions"
	assertionProjectExpr string = "s_id, s_created, s_fact, s_account, s_signature"
	assertionUpdateExpr  string = "SET " +
		"s_fact = :s_fact, " +
		"s_account = :s_account, " +
		"s_signature = :s_signature"

	assertionFactCreatedIndex            string = "s_fact-s_created-index"
	assertionFactCreatedIndexProjectExpr string = "s_id"
	assertionLoadByFactKeyCondExpr       string = "" +
		"s_fact = :s_fact AND"
)

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

// LoadAssertion loads a Assertion from its ID
func LoadAssertion(
	ctx context.Context,
	ID string,
) (*Assertion, error) {
	params := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"s_id": {S: aws.String(ID)},
		},
		TableName:            aws.String(assertionTableName),
		ConsistentRead:       aws.Bool(true),
		ProjectionExpression: aws.String(assertionProjectExpr),
	}
	resp, err := svc.GetItem(params)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if _, ok := resp.Item["s_id"]; !ok {
		return nil, nil
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
func (a *Assertion) Save(
	ctx context.Context,
) error {
	params := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"s_id":      {S: aws.String(a.ID)},
			"s_created": {N: aws.String(fmt.Sprintf("%d", a.Created))},
		},
		TableName: aws.String(assertionTableName),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":s_fact":      {S: aws.String(a.Fact)},
			":s_account":   {S: aws.String(string(a.Account))},
			":s_signature": {S: aws.String(string(a.Signature))},
		},
		UpdateExpression: aws.String(assertionUpdateExpr),
	}
	_, err := svc.UpdateItem(params)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// LoadAssertionsByFact loads all assertions related to a fact ordered by
// created date.
func LoadAssertionsByFact(
	ctx context.Context,
	fact string,
) ([]*Assertion, error) {
	params := &dynamodb.QueryInput{
		TableName:              aws.String(assertionTableName),
		IndexName:              aws.String(assertionFactCreatedIndex),
		ProjectionExpression:   aws.String(assertionFactCreatedIndexProjectExpr),
		KeyConditionExpression: aws.String(assertionLoadByFactKeyCondExpr),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":s_fact": {S: aws.String(string(fact))},
		},
		ScanIndexForward: aws.Bool(false),
	}

	resp, err := svc.Query(params)
	if err != nil {
		return nil, errors.Trace(err)
	}

	assertions := []*Assertion{}
	for _, it := range *resp.Items {
		a, err := LoadAssertion(ctx, *it["s_id"].S)
		if err != nil {
			return nil, errors.Trace(err)
		}
		assertions = append(facts, a)
	}
	return assertions, nil
}

// Verify verifies (in memory) that the assertion corresponds to the fact
// passed as argument and is properly signed.
func (a *Assertion) Verify(
	fact *Fact,
) bool {
	s, err := base64.StdEncoding.DecodeString(string(a.Signature))
	if err != nil {
		return false
	}

	from, err := keypair.Parse(string(a.Account))
	if err != nil {
		return false
	}

	err = from.Verify([]byte(fact.PayloadForAction(FaAssert)), s)
	if err != nil {
		return false
	}

	return true
}
