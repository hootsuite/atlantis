package dynamodb

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"strconv"
	"github.com/hootsuite/atlantis/models"
	"time"
)

type Backend struct {
	DB        dynamodbiface.DynamoDBAPI
	LockTable string
}

// dynamoLock duplicates the fields of models.ProjectLock and adds LocksKey
// so everything is a top-level field for serialization and then querying
// in DynamodB and also so any changes to models.ProjectLock won't affect
// how we're storing our data or will at least cause a compile error
type dynamoLock struct {
	LockKey string
	RepoFullName string
	Path string
	PullNum int
	Env     string
	Time    time.Time
}

func New(lockTable string, p client.ConfigProvider) Backend {
	return Backend{
		DB:        dynamodb.New(p),
		LockTable: lockTable,
	}
}

func (b Backend) key(project models.Project, env string) string {
	return fmt.Sprintf("%s/%s/%s", project.RepoFullName, project.Path, env)
}

func (b Backend) TryLock(project models.Project, env string, pullNum int) (bool, int, error) {
	key := b.key(project, env)
	newDynamoLock := dynamoLock{
		LockKey: key,
		RepoFullName: project.RepoFullName,
		Path: project.Path,
		PullNum: pullNum,
		Env:     env,
		Time:    time.Now(),
	}
	newLockSerialized, err := dynamodbattribute.MarshalMap(newDynamoLock)
	if err != nil {
		return false, 0, errors.Wrap(err, "serializing")
	}

	// check if there is an existing lock
	getItemParams := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"LockKey": {
				S: aws.String(key),
			},
		},
		TableName: aws.String(b.LockTable),
		ConsistentRead: aws.Bool(true),
	}
	item, err := b.DB.GetItem(getItemParams)
	if err != nil {
		return false, 0, errors.Wrap(err, "checking if lock exists")
	}

	// if there is already a lock then we can't acquire a lock. Return the existing lock
	var currLock dynamoLock
	if len(item.Item) != 0 {
		if err := dynamodbattribute.UnmarshalMap(item.Item, &currLock); err != nil {
			return false, 0, errors.Wrap(err,"found an existing lock at that key but it could not be deserialized. We suggest manually deleting this key from DynamoDB")
		}
		return false, currLock.PullNum, nil
	}

	// else we should be able to lock
	putItem := &dynamodb.PutItemInput{
		Item:      newLockSerialized,
		TableName: aws.String(b.LockTable),
		// this will ensure that we don't insert the new item in a race situation
		// where someone has written this key just after our read
		ConditionExpression: aws.String("attribute_not_exists(LockKey)"),
	}
	if _, err := b.DB.PutItem(putItem); err != nil {
		return false, 0, errors.Wrap(err, "writing lock")
	}
	return true, pullNum, nil
}

func (b Backend) Unlock(project models.Project, env string) error {
	key := b.key(project, env)
	params := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"LockKey": {S: aws.String(key)},
		},
		TableName: aws.String(b.LockTable),
	}
	_, err := b.DB.DeleteItem(params)
	return errors.Wrap(err, "deleting lock")
}

func (b Backend) List() ([]models.ProjectLock, error) {
	var locks []models.ProjectLock
	var err, internalErr error
	params := &dynamodb.ScanInput{
		TableName: aws.String(b.LockTable),
	}
	err = b.DB.ScanPages(params, func(out *dynamodb.ScanOutput, lastPage bool) bool {
		var dynamoLocks []dynamoLock
		if err := dynamodbattribute.UnmarshalListOfMaps(out.Items, &dynamoLocks); err != nil {
			internalErr = errors.Wrap(err,"deserializing locks")
			return false
		}
		for _, lock := range dynamoLocks {
			locks = append(locks, models.ProjectLock{
				PullNum: lock.PullNum,
				Project: models.NewProject(lock.RepoFullName, lock.Path),
				Env: lock.Env,
				Time: lock.Time,
			})
		}
		return lastPage
	})

	if err == nil && internalErr != nil {
		err = internalErr
	}
	return locks, errors.Wrap(err, "scanning dynamodb")
}

func (b Backend) UnlockByPull(repoFullName string, pullNum int) error {
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
		TableName: aws.String(b.LockTable),
	}

	// scan DynamoDB for locks that match the pull request
	var locks []dynamoLock
	var err, internalErr error
	err = b.DB.ScanPages(params, func(out *dynamodb.ScanOutput, lastPage bool) bool {
		if err := dynamodbattribute.UnmarshalListOfMaps(out.Items, &locks); err != nil {
			internalErr = errors.Wrap(err,"deserializing locks")
			return false
		}
		return lastPage
	})
	if err == nil {
		err = internalErr
	}
	if err != nil {
		return errors.Wrap(err, "scanning dynamodb")
	}

	// now we can unlock all of them
	for _, lock := range locks {
		if err := b.Unlock(models.NewProject(lock.RepoFullName, lock.Path), lock.Env); err != nil {
			return errors.Wrapf(err,"unlocking repo %s, path %s, env %s", lock.RepoFullName, lock.Path, lock.Env)
		}
	}
	return nil
}
