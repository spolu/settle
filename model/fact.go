package model

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/net/context"

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

const (
	factTableName   string = "facts"
	factProjectExpr string = "s_id, s_created, s_account, s_type, s_value"
	factUpdateExpr  string = "SET " +
		"s_created = :s_created, " +
		"s_type = :s_type, " +
		"s_value = :s_value"

	factAccountTypeIndex                string = "s_account-s_type-index"
	factAccountTypeIndexProjectExpr     string = "s_id"
	factLoadByAccountAndTypeKeyCondExpr string = "" +
		"s_account = :s_account AND " +
		"s_type = :s_type"
)

// NewFact creates a new Fact.
func NewFact(
	account PublicKey,
	t FctType,
	v string,
) *Fact {
	return &Fact{
		ID:      token.New("fact"),
		Created: time.Now().UnixNano(),
		Account: account,
		Type:    t,
		Value:   v,
	}
}

// LoadFact loads a Fact from its ID.
func LoadFact(
	ctx context.Context,
	account PublicKey,
	ID string,
) (*Fact, error) {
	params := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"s_account": {S: aws.String(string(account))},
			"s_id":      {S: aws.String(ID)},
		},
		TableName:            aws.String(factTableName),
		ConsistentRead:       aws.Bool(true),
		ProjectionExpression: aws.String(factProjectExpr),
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

	return &Fact{
		ID:      *resp.Item["s_id"].S,
		Created: created,
		Account: PublicKey(*resp.Item["s_account"].S),
		Type:    FctType(*resp.Item["s_type"].S),
		Value:   *resp.Item["s_value"].S,
	}, nil
}

// Save creates or updates the Fact.
func (f *Fact) Save(
	ctx context.Context,
) error {
	params := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"s_account": {S: aws.String(string(f.Account))},
			"s_id":      {S: aws.String(f.ID)},
		},
		TableName: aws.String(factTableName),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":s_created": {N: aws.String(fmt.Sprintf("%d", f.Created))},
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

// LoadFactsByAccountAndType loads a fact for a given account and type.
func LoadFactsByAccountAndType(
	ctx context.Context,
	account PublicKey,
	t FctType,
) ([]Fact, error) {
	params := &dynamodb.QueryInput{
		TableName:              aws.String(factTableName),
		IndexName:              aws.String(factAccountTypeIndex),
		ProjectionExpression:   aws.String(factAccountTypeIndexProjectExpr),
		KeyConditionExpression: aws.String(factLoadByAccountAndTypeKeyCondExpr),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":s_account": {S: aws.String(string(account))},
			":s_type":    {S: aws.String(string(t))},
		},
	}
	resp, err := svc.Query(params)
	if err != nil {
		return nil, errors.Trace(err)
	}

	facts := []Fact{}
	for _, it := range resp.Items {
		f, err := LoadFact(ctx, account, *it["s_id"].S)
		if err != nil {
			return nil, errors.Trace(err)
		} else if f == nil {
			return nil, errors.Newf(
				"Failed to load fact: %s", *it["s_id"].S)
		}
		facts = append(facts, *f)
	}
	return facts, nil
}

// LoadLatestFactByAccountAndType loads the most recent fact for a given
// account and type.
func LoadLatestFactByAccountAndType(
	ctx context.Context,
	account PublicKey,
	t FctType,
) (*Fact, error) {
	facts, err := LoadFactsByAccountAndType(ctx, account, t)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var latest *Fact
	for _, f := range facts {
		if latest == nil || f.Created > latest.Created {
			latest = &f
		}
	}
	return latest, nil
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
