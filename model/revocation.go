package model

import (
	"fmt"
	"strconv"
	"time"

	"golang.org/x/net/context"

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
	Account   PublicKey
	Signature PublicKeySignature
}

const (
	revocationTableName   string = "revocations"
	revocationProjectExpr string = "s_id, s_created, s_fact, s_account, s_signature"
	revocationUpdateExpr  string = "SET " +
		"s_fact = :s_fact, " +
		"s_account = :s_account, " +
		"s_signature = :s_signature"

	revocationFactCreatedIndex            string = "s_fact-s_created-index"
	revocationFactCreatedIndexProjectExpr string = "s_id"
	revocationLoadByFactKeyCondExpr       string = "" +
		"s_fact = :s_fact AND"
)

// NewRevocation creates a new revocation.
func NewRevocation(
	fact string,
	account PublicKey,
	signature PublicKeySignature,
) *Revocation {
	return &Revocation{
		ID:        token.New("revocation", string(account)),
		Created:   time.Now().UnixNano(),
		Fact:      fact,
		Account:   account,
		Signature: signature,
	}
}

// LoadRevocation loads a Revocation from its ID
func LoadRevocation(
	ID string,
) (*Revocation, error) {
	params := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"s_id": {S: aws.String(ID)},
		},
		TableName:            aws.String(revocationTableName),
		ConsistentRead:       aws.Bool(true),
		ProjectionExpression: aws.String(revocationProjectExpr),
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

	return &Revocation{
		ID:        ID,
		Created:   created,
		Fact:      *resp.Item["s_fact"].S,
		Account:   PublicKey(*resp.Item["s_account"].S),
		Signature: PublicKeySignature(*resp.Item["s_signature"].S),
	}, nil
}

// Save creates or updates the Revocation.
func (r *Revocation) Save() error {
	params := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"s_id":      {S: aws.String(r.ID)},
			"s_created": {N: aws.String(fmt.Sprintf("%d", r.Created))},
		},
		TableName: aws.String(revocationTableName),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":s_fact":      {S: aws.String(r.Fact)},
			":s_account":   {S: aws.String(string(r.Account))},
			":s_signature": {S: aws.String(string(r.Signature))},
		},
		UpdateExpression: aws.String(revocationUpdateExpr),
	}
	_, err := svc.UpdateItem(params)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// LoadRevocationsByFact loads all revocations related to a fact ordered by
// created date.
func LoadRevocationsByFact(
	ctx context.Context,
	fact string,
) ([]*Revocation, error) {
	params := &dynamodb.QueryInput{
		TableName:              aws.String(revocationTableName),
		IndexName:              aws.String(revocationFactCreatedIndex),
		ProjectionExpression:   aws.String(revocationFactCreatedIndexProjectExpr),
		KeyConditionExpression: aws.String(revocationLoadByFactKeyCondExpr),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":s_fact": {S: aws.String(string(fact))},
		},
		ScanIndexForward: aws.Bool(false),
	}

	resp, err := svc.Query(params)
	if err != nil {
		return nil, errors.Trace(err)
	}

	revocations := []*Revocation{}
	for _, it := range *resp.Items {
		a, err := LoadRevocation(ctx, *it["s_id"].S)
		if err != nil {
			return nil, errors.Trace(err)
		}
		revocations = append(facts, a)
	}
	return revocations, nil
}
