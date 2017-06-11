package locking

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	"encoding/json"
	"fmt"
	"encoding/hex"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"time"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"strconv"
)

// Data model:
// Key: LockID. LockID is made up of {repoFullName}/{path}/{environment}
// Queries that need to be run
// - get all locks
// - delete by ID
// - get locks for a specific pull num

type DynamoDBLockManager struct {
	DB        dynamodbiface.DynamoDBAPI
	LockTable string
}

type dynamoRun struct {
	LockID       string
	RepoFullName string
	Path         string
	Env          string
	PullNum      int
	User         string
	Timestamp    time.Time
}

func NewDynamoDBLockManager(lockTable string, p client.ConfigProvider) *DynamoDBLockManager {
	return &DynamoDBLockManager{
		DB:        dynamodb.New(p),
		LockTable: lockTable,
	}
}

func (d *DynamoDBLockManager) TryLock(run Run) (TryLockResponse, error) {
	var r TryLockResponse
	newRunSerialized, err := d.toDynamoItem(run)
	if err != nil {
		return r, errors.Wrap(err, "serializing")
	}

	// check if there is an existing lock
	getItemParams := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"LockID": {
				S: aws.String(run.StateKey()),
			},
		},
		TableName: aws.String(d.LockTable),
		ConsistentRead: aws.Bool(true),
	}
	item, err := d.DB.GetItem(getItemParams)
	if err != nil {
		return r, errors.Wrap(err, "checking if lock exists")
	}


	// if there is already a lock then we can't acquire a lock. Return the existing lock
	if len(item.Item) != 0 {
		var dynamoRun dynamoRun
		if err := dynamodbattribute.UnmarshalMap(item.Item, &dynamoRun); err != nil {
			return r, errors.Wrap(err,"found an existing lock at that id but it could not be deserialized. We suggest manually deleting this key from DynamoDB")
		}
		lockingRun := d.fromDynamoItem(dynamoRun)
		return TryLockResponse{
			LockAcquired: false,
			LockingRun: lockingRun,
			LockID: run.StateKey(),
		}, nil
	}

	// else we should be able to lock
	putItem := &dynamodb.PutItemInput{
		Item:      newRunSerialized,
		TableName: aws.String(d.LockTable),
		// this will ensure that we don't insert the new item in a race situation
		// where someone has written this key just after our read
		ConditionExpression: aws.String("attribute_not_exists(LockID)"),
	}
	if _, err := d.DB.PutItem(putItem); err != nil {
		return r, errors.Wrap(err, "writing lock")
	}
	return TryLockResponse{
		LockAcquired: true,
		LockingRun: run,
		LockID: run.StateKey(),
	}, nil
}

func (d *DynamoDBLockManager) Unlock(lockID string) error {
	params := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"LockID": {S: aws.String(lockID)},
		},
		TableName: aws.String(d.LockTable),
	}
	_, err := d.DB.DeleteItem(params)
	return errors.Wrap(err, "deleting lock")
}

func (d *DynamoDBLockManager) ListLocks() (map[string]Run, error) {
	params := &dynamodb.ScanInput{
		TableName: aws.String(d.LockTable),
	}

	m := make(map[string]Run)
	var err, internalErr error
	err = d.DB.ScanPages(params, func(out *dynamodb.ScanOutput, lastPage bool) bool {
		var runs []dynamoRun
		if err := dynamodbattribute.UnmarshalListOfMaps(out.Items, &runs); err != nil {
			internalErr = errors.Wrap(err,"deserializing locks")
			return false
		}
		for _, run := range runs {
			m[run.LockID] = d.fromDynamoItem(run)
		}
		return lastPage
	})

	if err == nil && internalErr != nil {
		err = internalErr
	}
	return m, errors.Wrap(err, "scanning dynamodb")
}

func (d *DynamoDBLockManager) FindLocksForPull(repoFullName string, pullNum int) ([]string, error) {
	params := &dynamodb.ScanInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pullNum": {
				N: aws.String(strconv.Itoa(pullNum)),
			},
			":repoFullName": {
				S: aws.String(repoFullName),
			},
		},
		FilterExpression: aws.String("RepoFullName = :repoFullName and PullNum = :pullNum"),
		TableName: aws.String(d.LockTable),
	}

	var ids []string
	var err, internalErr error
	err = d.DB.ScanPages(params, func(out *dynamodb.ScanOutput, lastPage bool) bool {
		var runs []dynamoRun
		if err := dynamodbattribute.UnmarshalListOfMaps(out.Items, &runs); err != nil {
			internalErr = errors.Wrap(err,"deserializing locks")
			return false
		}
		for _, run := range runs {
			ids = append(ids, run.LockID)
		}
		return lastPage
	})

	if err == nil && internalErr != nil {
		err = internalErr
	}
	return ids, errors.Wrap(err,"scanning dynamodb")
}

func (d *DynamoDBLockManager) deserializeItem(item map[string]*dynamodb.AttributeValue) (string, Run, error) {
	var lockID string
	var run Run

	lockIDItem, ok := item["LockID"]
	if !ok || lockIDItem == nil {
		return lockID, run, fmt.Errorf("lock did not have expected key 'LockID'")
	}
	lockID = string(hex.EncodeToString(lockIDItem.B))
	runItem, ok := item["Run"]
	if !ok || runItem == nil {
		return lockID, run, fmt.Errorf("lock did not have expected key 'Run'")
	}

	if err := d.deserialize(runItem.B, &run); err != nil {
		return lockID, run, fmt.Errorf("deserializing run at key %q: %s", lockID, err)
	}
	return lockID, run, nil
}

func (d *DynamoDBLockManager) deserialize(bs []byte, run *Run) error {
	return json.Unmarshal(bs, run)
}

func (d *DynamoDBLockManager) serialize(run Run) ([]byte, error) {
	return json.Marshal(run)
}

func (d *DynamoDBLockManager) toDynamoItem(run Run) (map[string]*dynamodb.AttributeValue, error) {
	item := dynamoRun{
		LockID: run.StateKey(),
		PullNum: run.PullNum,
		RepoFullName: run.RepoFullName,
		Env: run.Env,
		Path: run.Path,
		Timestamp: run.Timestamp,
		User: run.User,
	}
	return dynamodbattribute.MarshalMap(item)
}

func (d *DynamoDBLockManager) fromDynamoItem(dynamoRun dynamoRun) Run {
	return Run{
		User: dynamoRun.User,
		Timestamp: dynamoRun.Timestamp,
		Path: dynamoRun.Path,
		Env: dynamoRun.Env,
		RepoFullName: dynamoRun.RepoFullName,
		PullNum: dynamoRun.PullNum,
	}
}
