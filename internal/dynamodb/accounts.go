package dynamodb

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/rhpds/sandbox/internal/account"
	"github.com/rhpds/sandbox/internal/log"
	"go.uber.org/zap"
	"os"
	"strconv"
	"strings"
	"errors"
)

var svc *dynamodb.DynamoDB

var ErrAccountNotFound = errors.New("account not found")

// SetSession returns the current session
func SetSession() *dynamodb.DynamoDB {
	svc = dynamodb.New(session.New())
	return svc
}

// GetSession returns the current session
func GetSession() *dynamodb.DynamoDB {
	return svc
}

func parseNameInt(s string) int {
	var result strings.Builder
	for i := 0; i < len(s); i++ {
		b := s[i]
		if '0' <= b && b <= '9' {
			result.WriteByte(b)
		}
	}
	resultI, err := strconv.Atoi(result.String())
	if err != nil {
		log.Err.Fatal(err)
	}
	return resultI
}

// BuildAccounts returns the list of accounts from dynamodb scan output
func BuildAccounts(r *dynamodb.ScanOutput) []account.Account {
	accounts := []account.Account{}

	for _, sandbox := range r.Items {
		item := account.Account{}
		err := dynamodbattribute.UnmarshalMap(sandbox, &item)
		if err != nil {
			log.Logger.Fatal("Got error unmarshalling:", zap.Error(err))
		}

		item.NameInt = parseNameInt(item.Name)

		accounts = append(accounts, item)
	}

	return accounts
}


// GetAccount return an account from dynamodb
func GetAccount(name string) (account.Account, error) {
	sandbox := account.Account{Name: name}

	// Build the Get query input parameters
	input := &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("dynamodb_table")),
		Key: map[string]*dynamodb.AttributeValue{
			"name": {
				S: aws.String(name),
			},
		},
	}

	// Get the item from the table
	output, errget := svc.GetItem(input)

	if errget != nil {
		if aerr, ok := errget.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				log.Err.Println(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				log.Err.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeRequestLimitExceeded:
				log.Err.Println(dynamodb.ErrCodeRequestLimitExceeded, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				log.Err.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				log.Err.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Logger.Error(errget.Error())
		}

		log.Logger.Error("errget", zap.Error(errget))
		return account.Account{}, errget
	}

	if len(output.Item) == 0 {
		return account.Account{}, ErrAccountNotFound
	}

	if err := dynamodbattribute.UnmarshalMap(output.Item, &sandbox); err != nil {
		log.Logger.Fatal("Unmarshalling", zap.Error(err))
	}
	log.Logger.Info("GetItem succeeded", zap.String("sandbox", sandbox.Name))
	return sandbox, nil
}

// GetAccounts returns the list of accounts from dynamodb
func GetAccounts(filters []expression.ConditionBuilder) ([]account.Account, error) {
	accounts := []account.Account{}

	// Build dynamod query
	proj := expression.NamesList(
		expression.Name("name"),
		expression.Name("available"),
		expression.Name("to_cleanup"),
		expression.Name("guid"),
		expression.Name("service_uuid"),
		expression.Name("envtype"),
		expression.Name("owner"),
		expression.Name("zone"),
		expression.Name("hosted_zone_id"),
		expression.Name("account_id"),
		expression.Name("comment"),
		expression.Name("owner_email"),
		expression.Name("aws:rep:updatetime"),
		expression.Name("aws_access_key_id"),
		expression.Name("aws_secret_access_key"),
		expression.Name("conan_status"),
		expression.Name("conan_timestamp"),
		expression.Name("conan_hostname"),
	)

	builder := expression.NewBuilder()

	for _, filter := range filters {
		builder = builder.WithFilter(filter)
	}

	expr, err := builder.WithProjection(proj).Build()

	if err != nil {
		log.Err.Println("Got error building expression:")
		log.Err.Println(err.Error())
		os.Exit(1)
	}

	input := &dynamodb.ScanInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		TableName:                 aws.String(os.Getenv("dynamodb_table")),
		ProjectionExpression:      expr.Projection(),
		FilterExpression:          expr.Filter(),
	}

	errscan := svc.ScanPages(input,
		func(page *dynamodb.ScanOutput, lastPage bool) bool {
			accounts = append(accounts, BuildAccounts(page)...)
			return true
		})

	//result, err := svc.Scan(input)
	if errscan != nil {
		if aerr, ok := errscan.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				log.Err.Println(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				log.Err.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeRequestLimitExceeded:
				log.Err.Println(dynamodb.ErrCodeRequestLimitExceeded, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				log.Err.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				log.Err.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Err.Println(errscan.Error())
		}
		return []account.Account{}, errscan
	}

	return accounts, nil
}
