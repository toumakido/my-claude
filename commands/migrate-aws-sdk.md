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
   - Pagination patterns (ScanPages/QueryPages → Paginator)
   - Expression parameter types (ExpressionAttributeNames/Values)
   - Enum type comparisons (remove pointer dereference)
   - Setter methods → direct field assignment
   - Marshal/Unmarshal package changes
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
   - DynamoDB: add attributevalue package: `go get github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue`
   - Run `go mod tidy`
   - If submodules exist (independent go.mod files in subdirectories):
     - Update dependencies in each submodule directory (e.g., `cd lambda/ses-notification && go get github.com/aws/aws-sdk-go-v2/...`)
     - Run `go mod tidy` in each submodule
     - Run `go build` in each submodule to verify
6. Verify compilation and context usage:
   - Root directory: `go build ./...`
   - Each submodule: `go build .`
   - Run tests if available: `go test ./...`
   - Check for inappropriate context.Background() usage: `grep -r "context.Background()" --include="*.go" | grep -v "func main\|func init\|_test.go"`
   - Verify no type errors (especially enum types, types.X usage)
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

### Types Package Import

Many services require explicit import of `types` package separately from service package:

```go
// Import types package as needed
import (
    "github.com/aws/aws-sdk-go-v2/service/ses"
    sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"

    "github.com/aws/aws-sdk-go-v2/service/sesv2"
    sesv2types "github.com/aws/aws-sdk-go-v2/service/sesv2/types"

    "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
    idptypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)
```

Types package is needed for:
- Request/Response struct field types (Body, Message, Content, etc.)
- Enum type constants (AuthFlowType, SuppressionListReason, etc.)
- Data types like AttributeValue

## Service-Specific Patterns

### SES/SESv2

#### Address Lists Type Change
```go
// v1
to := []*string{}
cc := []*string{}
replyTo := []*string{}
for _, obj := range recipients {
    to = append(to, obj.Address())  // *string
}

// v2
to := []string{}
cc := []string{}
replyTo := []string{}
for _, obj := range recipients {
    to = append(to, swag.StringValue(obj.Address()))  // string
}

// v2 API call
client.SendEmail(ctx, &ses.SendEmailInput{
    Message: &sestypes.Message{
        Subject: &sestypes.Content{...},
        Body: body,
    },
    Source: mail.From.Address(),
    Destination: &sestypes.Destination{
        ToAddresses:  to,      // []string
        CcAddresses:  cc,      // []string
        BccAddresses: bcc,     // []string
    },
    ReplyToAddresses: replyTo,  // []string
})
```

#### SESv2 Type Constants
```go
// v1
import "github.com/aws/aws-sdk-go/service/sesv2"

input := &sesv2.PutSuppressedDestinationInput{
    Reason: aws.String(sesv2.SuppressionListReasonBounce),
    EmailAddress: aws.String(email),
}

// v2
import (
    "github.com/aws/aws-sdk-go-v2/service/sesv2"
    sesv2types "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

input := &sesv2.PutSuppressedDestinationInput{
    Reason: sesv2types.SuppressionListReasonBounce,  // use type constant
    EmailAddress: aws.String(email),
}
```

#### Types Package Import
```go
// Required: import both ses and sestypes
import (
    "github.com/aws/aws-sdk-go-v2/service/ses"
    sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
)
```

### Cognito

#### AuthFlow and AuthParameters Type Change
```go
// v1
import idp "github.com/aws/aws-sdk-go/service/cognitoidentityprovider"

output, err := idp.New(sess).InitiateAuthWithContext(ctx, &idp.InitiateAuthInput{
    ClientId: aws.String(clientID),
    AuthFlow: aws.String("USER_PASSWORD_AUTH"),
    AuthParameters: map[string]*string{
        "USERNAME": aws.String(username),
        "PASSWORD": aws.String(password),
    },
})

// v2
import (
    idp "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
    idptypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

client := idp.NewFromConfig(cfg)
authFlow := idptypes.AuthFlowTypeUserPasswordAuth
output, err := client.InitiateAuth(ctx, &idp.InitiateAuthInput{
    ClientId: aws.String(clientID),
    AuthFlow: authFlow,  // enum type
    AuthParameters: map[string]string{  // no pointers
        "USERNAME": username,
        "PASSWORD": password,
    },
})
```

#### AuthenticationResultType Type Change
```go
// v1
result *idp.AuthenticationResultType

// v2
result *idptypes.AuthenticationResultType
```

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
import (
    "github.com/aws/aws-sdk-go/service/dynamodb"
    "github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var DynamoClient *dynamodb.DynamoDB

// AttributeValue
Item: map[string]*dynamodb.AttributeValue{
    "Key": {S: aws.String("value")},
}

// Expression parameters
ExpressionAttributeNames: map[string]*string{
    "#Key": aws.String("RealKey"),
}
ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
    ":val": {S: aws.String("value")},
}

// Setter methods
input.SetProjectionExpression("Key1,Key2")

// KeyType comparison
if keyElement.KeyType != nil && *keyElement.KeyType == dynamodb.KeyTypeHash {
    // process
}

// Marshal/Unmarshal
dynamodbattribute.UnmarshalMap(resp.Item, &result)
dynamodbattribute.MarshalMap(item)

// Pagination
err = client.ScanPages(input,
    func(page *dynamodb.ScanOutput, lastPage bool) bool {
        for _, item := range page.Items {
            // process
        }
        return len(page.LastEvaluatedKey) > 0
    })

err = client.QueryPages(input,
    func(page *dynamodb.QueryOutput, lastPage bool) bool {
        for _, item := range page.Items {
            // process
        }
        return !lastPage
    })

// v2
import (
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
    "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
)

var DynamoClient *dynamodb.Client

// AttributeValue (type changed)
Item: map[string]types.AttributeValue{
    "Key": &types.AttributeValueMemberS{Value: "value"},
}

// Expression parameters (no pointers)
ExpressionAttributeNames: map[string]string{
    "#Key": "RealKey",
}
ExpressionAttributeValues: map[string]types.AttributeValue{
    ":val": &types.AttributeValueMemberS{Value: "value"},
}

// Direct field assignment
projExpr := "Key1,Key2"
input.ProjectionExpression = &projExpr

// KeyType comparison (no pointer dereference)
if keyElement.KeyType == types.KeyTypeHash {
    // process
}

// Marshal/Unmarshal (package changed)
attributevalue.UnmarshalMap(resp.Item, &result)
attributevalue.MarshalMap(item)

// Pagination with Paginator
paginator := dynamodb.NewScanPaginator(client, input)
for paginator.HasMorePages() {
    page, err := paginator.NextPage(ctx)
    if err != nil {
        // handle error
        break
    }
    for _, item := range page.Items {
        // process
    }
}

queryPaginator := dynamodb.NewQueryPaginator(client, queryInput)
for queryPaginator.HasMorePages() {
    page, err := queryPaginator.NextPage(ctx)
    if err != nil {
        break
    }
    for _, item := range page.Items {
        // process
    }
}
```

## X-Ray Instrumentation

### v2 SDK Support
```go
// v1
import "github.com/aws/aws-xray-sdk-go/xray"

sesClient := sesv2.New(session.Must(session.NewSession()))
xray.AWS(sesClient.Client)

// v2
import (
    "github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
    "github.com/aws/aws-xray-sdk-go/xray"
)

cfg, err := config.LoadDefaultConfig(ctx)
if err != nil {
    return err
}
awsv2.AWSV2Instrumentor(&cfg.APIOptions)
sesClient := sesv2.NewFromConfig(cfg)
```

## Context Usage Guidelines

- **handler/controller layer**: Obtain context from web framework
- **usecase/service layer**: Receive as parameter, pass to lower layers
- **repository/infrastructure layer**: Receive as parameter, use for AWS API calls
- **test/fake implementations**: Use `context.TODO()` or `context.Background()`
- **main function/initialization**: Use `context.Background()` or `context.TODO()`

```go
// Test/fake implementation example
func (f *FakeService) Cleanup() {
    ctx := context.TODO()
    resp, err := f.client.ListTables(ctx, &dynamodb.ListTablesInput{})
    // ...
}

// Repository layer example
func (r *Repository) GetItem(ctx context.Context, id string) (Item, error) {
    result, err := r.dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
        TableName: aws.String(r.tableName),
        Key: map[string]types.AttributeValue{
            "ID": &types.AttributeValueMemberS{Value: id},
        },
    })
    // ...
}
```

## Web Framework Integration

### Echo Framework

```go
// Obtain context from Echo in handler layer
func (h *Handler) GetItem(c echo.Context) error {
    ctx := c.Request().Context()

    // Pass to repository/service layer
    result, err := h.repository.GetItem(ctx, id)
    // ...
}
```

## Notes

- Backup or commit before running migration
- Test thoroughly after migration - some APIs have behavioral changes
- Check AWS SDK Go v2 migration guide for service-specific changes
- Update unit tests to match new patterns
- Consider using existing context.Context from caller functions instead of context.TODO()
