package model

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/spolu/settl/util/errors"
	"github.com/spolu/settl/util/token"
)

// Fact represents the storage model for a fact.
type Fact struct {
	ID      string
	Created int64
	Account PublicKey
	Type    FctType
	Value   string
}

var factProjectExpr = "s_id, s_created, s_account, s_type, s_value"
var factUpdateExpr = "SET " +
	"s_created = :s_created, " +
	"s_account = :s_account, " +
	"s_type = :s_type, " +
	"s_value = :s_value"
var factTableName = "facts"

// NewFact creates a new Fact.
func NewFact(
	account PublicKey,
	t FctType,
	v string,
) *Fact {
	return &Fact{
		ID:      token.New("fact", string(account)),
		Created: time.Now().UnixNano(),
		Account: account,
		Type:    t,
		Value:   v,
	}
}

// LoadFact loads a Fact from its ID.
func LoadFact(
	ID string,
) (*Fact, error) {
	params := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"s_id": {S: aws.String(ID)},
		},
		TableName:            aws.String(factTableName),
		ConsistentRead:       aws.Bool(true),
		ProjectionExpression: aws.String(factProjectExpr),
	}
	resp, err := svc.GetItem(params)
	if err != nil {
		return nil, errors.Trace(err)
	}

	created, err := strconv.ParseInt(*resp.Item["s_created"].N, 10, 64)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &Fact{
		ID:      ID,
		Created: created,
		Account: PublicKey(*resp.Item["s_account"].S),
		Type:    FctType(*resp.Item["s_type"].S),
		Value:   *resp.Item["s_value"].S,
	}, nil
}

// Save creates or updates the Fact.
func (f *Fact) Save() error {
	params := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"s_id": {S: aws.String(f.ID)},
		},
		TableName: aws.String(factTableName),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":s_created": {N: aws.String(fmt.Sprintf("%d", f.Created))},
			":s_account": {S: aws.String(string(f.Account))},
			":s_type":    {S: aws.String(string(f.Type))},
			":s_value":   {S: aws.String(f.Value)},
		},
		UpdateExpression: aws.String(factUpdateExpr),
	}
	_, err := svc.UpdateItem(params)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// PayloadForAction constructs the payload to be signed for a particular action
// related to a fact.
func (f *Fact) PayloadForAction(
	action FctAction,
) string {
	payload := url.Values{}
	payload.Set("action", string(action))
	payload.Set("account", string(f.Account))
	payload.Set("type", string(f.Type))
	payload.Set("value", string(f.Value))

	return payload.Encode()
}
