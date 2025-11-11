Migrate aws-sdk-go-v1 to aws-sdk-go-v2 in current repository

Output language: Japanese, formal business tone

## Prerequisites

- Current repository uses aws-sdk-go-v1
- go.mod exists in repository
- Git working tree should be clean (recommended)

## Process

1. Search for aws-sdk-go-v1 usage: `grep -r "github.com/aws/aws-sdk-go" --include="*.go"`
2. Create TodoWrite plan with all files to migrate and migration steps
3. Analyze each file for migration patterns:
   - Session/Config initialization
   - Service client creation patterns
   - API call syntax and context usage
   - Context propagation in Repository/Service layers
   - Error handling
4. Execute automatic migration for each file:
   - Update import statements
   - Replace session initialization with config.LoadDefaultConfig
   - Update service client creation
   - Add context parameters to API calls (propagate from caller, avoid context.Background() in non-entry points)
   - Update interfaces to include context.Context parameters
   - Preserve existing logic and error handling
5. Update go.mod dependencies:
   - Add v2 dependencies: `go get github.com/aws/aws-sdk-go-v2/config`
   - Add required service packages
   - Run `go mod tidy`
6. Verify compilation and context usage:
   - `go build ./...`
   - Check for inappropriate context.Background() usage: `grep -r "context.Background()" --include="*.go" | grep -v "func main\|func init"`
7. Report migration summary with file count and changes

## Migration Patterns

### Session → Config
```go
// v1
sess := session.Must(session.NewSession())

// v2
cfg, err := config.LoadDefaultConfig(context.TODO())
if err != nil {
    // handle error
}
```

### Service Client Initialization
```go
// v1
svc := s3.New(sess)

// v2
svc := s3.NewFromConfig(cfg)
```

### API Calls with Context
```go
// v1
result, err := svc.GetObject(&s3.GetObjectInput{...})

// v2
result, err := svc.GetObject(context.TODO(), &s3.GetObjectInput{...})
```

### Context Propagation in Repository/Service Layer
```go
// v1
func (repo *Repo) Put(item Item) error {
    _, err := repo.dynamo.PutItem(&dynamodb.PutItemInput{...})
    return err
}

// v2 - avoid creating context.Background() in non-entry points
func (repo *Repo) Put(ctx context.Context, item Item) error {
    _, err := repo.dynamo.PutItem(ctx, &dynamodb.PutItemInput{...})
    return err
}

// Interface update required
type Repository interface {
    Put(context.Context, Item) error
}
```

### Import Paths
```go
// v1
import (
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
)

// v2
import (
    "context"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)
```

## Service-Specific Patterns

### SQS
```go
// v1
MaxNumberOfMessages: aws.Int64(10)
MessageAttributeNames: []*string{aws.String("UserID"), aws.String("RPID")}

// v2
MaxNumberOfMessages: aws.Int32(10)  // or just: 10
MessageAttributeNames: []string{"UserID", "RPID"}
```

### DynamoDB
```go
// v1
import "github.com/aws/aws-sdk-go/service/dynamodb"

var DynamoClient *dynamodb.DynamoDB
Item: map[string]*dynamodb.AttributeValue{
    "Key": {S: aws.String("value")},
}

// v2
import (
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var DynamoClient *dynamodb.Client
Item: map[string]types.AttributeValue{
    "Key": &types.AttributeValueMemberS{Value: "value"},
}
```

## Notes

- Backup or commit before running migration
- Test thoroughly after migration - some APIs have behavioral changes
- Check AWS SDK Go v2 migration guide for service-specific changes
- Update unit tests to match new patterns
- Consider using existing context.Context from caller functions instead of context.TODO()
