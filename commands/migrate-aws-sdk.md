Migrate aws-sdk-go-v1 to aws-sdk-go-v2 in current repository

Output language: Japanese, formal business tone

## Prerequisites

- Current repository uses aws-sdk-go-v1
- go.mod exists in repository
- Git working tree should be clean (recommended)

## Process

1. Search for aws-sdk-go-v1 usage: `grep -r "github.com/aws/aws-sdk-go" --include="*.go"`
2. Create TodoWrite plan with all files to migrate and migration steps
3. Analyze each file for migration patterns by scanning in this order:
   - File type detection: main.go, handler files, repository files, service files, test files
   - Session/Config initialization
   - Service client creation patterns
   - API call syntax and context usage
   - Context propagation patterns:
     * Handler layer: Echo handlers `func (h *Handler) Method(c echo.Context)`, Lambda handlers `func handler(ctx context.Context, event XXX)`
     * Helper functions: Top-level functions in main.go hierarchy that call AWS SDK (no receiver)
     * Service/Repository layer: Methods with receivers `func (s *Service) Method(...)`
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
   - Add context parameters in this order:
     1. Handler layer functions/methods (process all handler patterns first)
     2. Helper functions that call AWS SDK (top-level functions without receiver)
     3. Repository/Service methods
     4. Update all call sites for modified functions/methods

     Apply context handling rules:
     * Client initialization (init/main/NewXxx functions): use context.Background()
     * Handler layer (Echo): obtain context with `ctx := c.Request().Context()`, pass to downstream
     * Handler layer (Lambda): use handler's ctx parameter directly, pass to helper functions
     * Helper functions: add `ctx context.Context` as first parameter, receive from caller
     * Repository/Service methods: add `ctx context.Context` as first parameter after receiver, propagate from caller
     * Never use context.TODO() in production code (tests only)
     * Store clients in struct fields (dependency injection), never reinitialize per method call
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
   - Check for inappropriate context usage:
     * `grep -r "context\.TODO()\|context\.Background()" --include="*.go" | grep -v "func init\|func main\|func New" | grep -v "_test\.go" | grep -v "^[[:space:]]*//"`
     * Look for helper functions missing ctx parameter: `grep -r "^func [a-z]" --include="*.go" | grep -v "_test\.go"` and verify each has ctx parameter
     * Verify all AWS SDK calls receive context as first parameter
   - For Lambda functions: verify context propagation from handler to all helper functions and AWS API calls
   - For Echo handlers: verify `ctx := c.Request().Context()` at top of handler methods
   - Verify no type errors (especially enum types, types.X usage)
7. Report migration summary with file count and changes

## Migration Patterns

### Session → Config
```go
// v1
sess := session.Must(session.NewSession())

// v2 (in init/main/New* functions)
cfg, err := config.LoadDefaultConfig(context.Background())
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

// v2 (use context from caller)
result, err := svc.GetObject(ctx, &s3.GetObjectInput{...})
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

### ECS

#### RunTaskInput Type Changes
```go
// v1
import (
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/service/ecs"
)

client := ecs.New(session)
input := &ecs.RunTaskInput{
    Cluster:        aws.String("cluster-name"),
    Count:          aws.Int64(1),
    TaskDefinition: aws.String("task-def"),
    LaunchType:     aws.String(ecs.LaunchTypeFargate),
    NetworkConfiguration: &ecs.NetworkConfiguration{
        AwsvpcConfiguration: &ecs.AwsVpcConfiguration{
            SecurityGroups: aws.StringSlice([]string{"sg-xxx"}),
            Subnets:        aws.StringSlice([]string{"subnet-xxx"}),
            AssignPublicIp: aws.String(ecs.AssignPublicIpDisabled),
        },
    },
}

// v2
import (
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/ecs"
    ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

cfg, err := config.LoadDefaultConfig(ctx)
client := ecs.NewFromConfig(cfg)

count := int32(1)
input := &ecs.RunTaskInput{
    Cluster:        aws.String("cluster-name"),
    Count:          &count,  // *int32
    TaskDefinition: aws.String("task-def"),
    LaunchType:     ecstypes.LaunchTypeFargate,  // enum type
    NetworkConfiguration: &ecstypes.NetworkConfiguration{
        AwsvpcConfiguration: &ecstypes.AwsVpcConfiguration{
            SecurityGroups: []string{"sg-xxx"},  // no pointer slice
            Subnets:        []string{"subnet-xxx"},
            AssignPublicIp: ecstypes.AssignPublicIpDisabled,  // enum type
        },
    },
}

_, err = client.RunTask(ctx, input)
```

Key changes:
- `Count`: `*int64` → `*int32`
- `LaunchType`: `*string` → `ecstypes.LaunchType` (enum)
- `SecurityGroups`/`Subnets`: `[]*string` → `[]string`
- `AssignPublicIp`: `*string` → `ecstypes.AssignPublicIp` (enum)

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

#### Local Endpoint Configuration

For testing with local DynamoDB, use one of the following recommended patterns:

```go
// v1 (deprecated pattern - do not use)
cfg, err := config.LoadDefaultConfig(ctx,
    config.WithEndpointResolverWithOptions(
        aws.EndpointResolverWithOptionsFunc(
            func(service, region string, options ...interface{}) (aws.Endpoint, error) {
                return aws.Endpoint{URL: endpoint}, nil
            },
        ),
    ),
)

// v2 - Method 1: Service-specific BaseEndpoint (recommended for DynamoDB)
cfg, err := config.LoadDefaultConfig(ctx)
if err != nil {
    // handle error
}
client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
    o.BaseEndpoint = aws.String("http://localhost:8000")
})

// v2 - Method 2: EndpointResolverV2
cfg, err := config.LoadDefaultConfig(ctx)
if err != nil {
    // handle error
}
resolver := dynamodb.EndpointResolverFromURL("http://localhost:8000")
client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
    o.EndpointResolverV2 = resolver
})

// v2 - Method 3: Global config.WithBaseEndpoint (affects all services)
cfg, err := config.LoadDefaultConfig(
    ctx,
    config.WithBaseEndpoint("http://localhost:8000"),
)
if err != nil {
    // handle error
}
client := dynamodb.NewFromConfig(cfg)
```

**Note**: `WithEndpointResolverWithOptions` is deprecated. For DynamoDB local testing, Method 1 (service-specific `BaseEndpoint`) is recommended as it only affects the DynamoDB client without impacting other AWS services.

#### Error Handling - Type-safe Approach

DynamoDB operations should use type-safe error checking instead of string comparison.

```go
// v1 - String comparison (not type-safe)
var apiErr smithy.APIError
if ok := errors.As(err, &apiErr); ok {
    if apiErr.ErrorCode() == "ProvisionedThroughputExceededException" {
        // Typo risk, no compile-time check
        continue
    }
}

// v2 - Type-safe error checking (recommended)
import (
    "errors"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var pte *types.ProvisionedThroughputExceededException
if errors.As(err, &pte) {
    // Handle throughput exceeded
    // Compile-time type checking, IDE autocomplete support
}

var rnfe *types.ResourceNotFoundException
if errors.As(err, &rnfe) {
    // Handle not found
}

var cve *types.ConditionalCheckFailedException
if errors.As(err, &cve) {
    // Handle conditional check failure
}
```

**Benefits**:
- Compile-time type checking
- No typo risk in error code strings
- Better IDE support (autocomplete, refactoring)
- More explicit error handling

#### UpdateItem with Dynamic Expression Building

For complex updates with conditional fields:

```go
// v2 - Dynamic UpdateExpression with conditional fields
import "strings"

func buildUpdateExpression(data UpdateData) (
    ean map[string]string,
    eav map[string]types.AttributeValue,
    updateExpr string,
) {
    ean = map[string]string{}
    eav = map[string]types.AttributeValue{}
    updateList := []string{}
    removeList := []string{}

    // Conditional field updates
    if data.Name != nil {
        ean["#Name"] = "Name"
        eav[":Name"] = &types.AttributeValueMemberS{Value: *data.Name}
        updateList = append(updateList, "#Name=:Name")
    } else {
        removeList = append(removeList, "#Name")
    }

    if data.Count != nil {
        ean["#Count"] = "Count"
        eav[":Count"] = &types.AttributeValueMemberN{Value: strconv.Itoa(*data.Count)}
        updateList = append(updateList, "#Count=:Count")
    }

    // Build expression
    updateExpr = "SET " + strings.Join(updateList, ",")
    if len(removeList) > 0 {
        updateExpr += " REMOVE " + strings.Join(removeList, ",")
    }

    return
}

// Usage
ean, eav, updateExpr := buildUpdateExpression(data)
input := &dynamodb.UpdateItemInput{
    Key: map[string]types.AttributeValue{
        "ID": &types.AttributeValueMemberS{Value: id},
    },
    TableName:                 aws.String(tableName),
    ExpressionAttributeNames:  ean,
    ExpressionAttributeValues: eav,
    UpdateExpression:          &updateExpr,
}
_, err := client.UpdateItem(ctx, input)
```

#### Testing with AttributeValue Type Assertions

```go
// v1 - Direct field access
if a := result["Code"]; assert.NotNil(t, a) {
    if v := a.S; assert.NotNil(t, v) {
        assert.Equal(t, "AAPL", *v)
    }
}

// v2 - Type assertion pattern (required)
if a := result["Code"]; assert.NotNil(t, a) {
    if v, ok := a.(*types.AttributeValueMemberS); assert.True(t, ok) {
        assert.Equal(t, "AAPL", v.Value)
    }
}

// v2 - Sorting results with type assertion
sort.SliceStable(results, func(i, j int) bool {
    codeI, okI := results[i]["Code"].(*types.AttributeValueMemberS)
    codeJ, okJ := results[j]["Code"].(*types.AttributeValueMemberS)
    if okI && okJ {
        return codeI.Value < codeJ.Value
    }
    return false
})
```

#### Enum Types for Request Parameters

```go
// v1 - String pointer
ReturnConsumedCapacity: aws.String(dynamodb.ReturnConsumedCapacityNone)

// v2 - Enum type constant
ReturnConsumedCapacity: types.ReturnConsumedCapacityNone

// v1 - String pointer for ReturnValues
ReturnValues: aws.String(dynamodb.ReturnValueAllNew)

// v2 - Enum type
ReturnValues: types.ReturnValueAllNew
```

#### Retry Logic Best Practices

When implementing retry logic for throughput exceptions, always include proper limits and backoff:

```go
import (
    "errors"
    "math"
    "time"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const maxRetries = 3

func scanWithRetry(ctx context.Context, client *dynamodb.Client, input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error) {
    retryCount := 0

    for {
        output, err := client.Scan(ctx, input)
        if err != nil {
            var pte *types.ProvisionedThroughputExceededException
            if errors.As(err, &pte) {
                if retryCount >= maxRetries {
                    return nil, fmt.Errorf("max retries exceeded: %w", err)
                }
                retryCount++
                backoff := time.Second * time.Duration(math.Pow(2, float64(retryCount)))

                // Check context deadline
                select {
                case <-ctx.Done():
                    return nil, ctx.Err()
                case <-time.After(backoff):
                    continue
                }
            }
            return nil, err
        }
        return output, nil
    }
}
```

**Key Points**:
- Always set maximum retry count to prevent infinite loops
- Use exponential backoff for retry delays
- Check context deadline to respect timeout
- Never use infinite retry loops in production

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

Apply phases in order to ensure proper context initialization and propagation:

### Phase 1: Identify context initialization points (use context.Background())
Functions that create new contexts from scratch:
- `func init()` - application initialization
- `func main()` - application entry point
- `func NewXxx()` - when calling config.LoadDefaultConfig
- Never initialize context inside Repository/Service/Helper functions

### Phase 2: Identify context entry points (where context enters the call chain)
Functions that receive context from external sources:
- **Echo handler**: `ctx := c.Request().Context()` at the top of handler method
- **Lambda handler**: `ctx` parameter from function signature `func handler(ctx context.Context, event XXX)`

### Phase 3: Add context propagation (add ctx parameter, receive from caller)
Functions that pass context through the call chain:
- **Helper functions** (top-level functions in main.go hierarchy): add `ctx context.Context` as first parameter
- **Repository methods**: add `ctx context.Context` as first parameter after receiver
- **Service methods**: add `ctx context.Context` as first parameter after receiver

### Phase 4: Pass context to all downstream calls
All AWS SDK calls and function calls must receive context:
- AWS SDK calls: `client.Method(ctx, input)`
- Function/method calls: `functionName(ctx, ...)`

### Pattern matching rules (apply in order)
1. `func init()` or `func main()` → use `context.Background()`
2. `func NewRepository()` or `func NewService()` or `func NewHandler()` → use `context.Background()` for config.LoadDefaultConfig
3. Handler layer patterns:
   - `func (h *Handler) Method(c echo.Context)` → add `ctx := c.Request().Context()` at top, pass ctx to downstream
   - `func handler(ctx context.Context, event XXXEvent)` → pass ctx to all helper functions and AWS SDK calls
4. Helper functions (top-level functions calling AWS SDK):
   - `func processXXX(...)` → add `ctx context.Context` as first param, update all call sites
5. `func (r *Repository) Method(...)` or `func (s *Service) Method(...)` → add `ctx context.Context` as first param after receiver
6. Inside `*_test.go` → `context.TODO()` or `context.Background()` acceptable
7. Type name matches `*Fake*`, `*Mock*`, `*Stub*` → `context.TODO()` or `context.Background()` acceptable

Examples:

```go
// Pattern 1: Client initialization in constructor
func NewRepository() (*Repository, error) {
    cfg, err := config.LoadDefaultConfig(context.Background())
    if err != nil {
        return nil, err
    }
    return &Repository{
        dynamoClient: dynamodb.NewFromConfig(cfg),
    }, nil
}

// Pattern 3: Repository method with context propagation
func (r *Repository) Get(ctx context.Context, id string) error {
    _, err := r.dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{...})
    return err
}

// Pattern 3: Service method propagating context
func (s *Service) Process(ctx context.Context, id string) error {
    return s.repo.Get(ctx, id)
}

// Wrong: Reinitializing client in method (violates rule 1)
func (r *Repository) Get(ctx context.Context, id string) error {
    cfg, _ := config.LoadDefaultConfig(context.Background())
    client := dynamodb.NewFromConfig(cfg)  // DO NOT DO THIS
    return nil
}

// Pattern 4: Helper function in main.go hierarchy (Lambda example)
// Before migration
func processRecord(bucket, key string) error {
    _, err := dynamoClient.PutItem(&dynamodb.PutItemInput{...})
    return err
}

func handler(ctx context.Context, event events.S3Event) error {
    for _, record := range event.Records {
        if err := processRecord(record.S3.Bucket.Name, record.S3.Object.Key); err != nil {
            return err
        }
    }
    return nil
}

// After migration - add ctx parameter to helper function and pass from handler
func processRecord(ctx context.Context, bucket, key string) error {
    _, err := dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{...})
    return err
}

func handler(ctx context.Context, event events.S3Event) error {
    for _, record := range event.Records {
        if err := processRecord(ctx, record.S3.Bucket.Name, record.S3.Object.Key); err != nil {
            return err
        }
    }
    return nil
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

## Lambda Functions

### Context Propagation

Lambda functions must propagate context from handler to all AWS API calls.

```go
// v1
func handler(ctx context.Context, event events.S3Event) error {
    for _, record := range event.Records {
        if err := processRecord(record.S3.Bucket.Name, record.S3.Object.Key); err != nil {
            return err
        }
    }
    return nil
}

func processRecord(bucket, key string) error {
    _, err := dynamoClient.PutItem(&dynamodb.PutItemInput{...})
    return err
}

// v2
func handler(ctx context.Context, event events.S3Event) error {
    for _, record := range event.Records {
        if err := processRecord(ctx, record.S3.Bucket.Name, record.S3.Object.Key); err != nil {
            return err
        }
    }
    return nil
}

func processRecord(ctx context.Context, bucket, key string) error {
    _, err := dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{...})
    return err
}
```

Never use `context.TODO()` or `context.Background()` in Lambda functions. Always propagate the context from handler to enable proper timeout handling, X-Ray tracing, and cancellation signal propagation.

### Config Initialization in init()

For global client variables, use init function for config initialization.

```go
// v1
var (
    dynamoClient dynamodbiface.DynamoDBAPI
    ecsClient    ecsiface.ECSAPI
)

func init() {
    sess := session.Must(session.NewSession(
        aws.NewConfig().WithRegion("ap-northeast-1"),
    ))
    dynamoClient = dynamodb.New(sess)
    ecsClient = ecs.New(sess)
}

// v2
var (
    dynamoClient *dynamodb.Client
    ecsClient    *ecs.Client
)

func init() {
    cfg, err := config.LoadDefaultConfig(
        context.Background(),
        config.WithRegion("ap-northeast-1"),
    )
    if err != nil {
        log.Fatalf("unable to load SDK config, %v", err)
    }
    dynamoClient = dynamodb.NewFromConfig(cfg)
    ecsClient = ecs.NewFromConfig(cfg)
}
```

Use `context.Background()` in init function (entry point), but use caller's context for actual API calls.

## Notes

- Backup or commit before running migration
- Test thoroughly after migration - some APIs have behavioral changes
- Check AWS SDK Go v2 migration guide for service-specific changes
- Update unit tests to match new patterns
- Follow Context Usage Guidelines: use context.Background() for initialization, propagate caller's context for API calls
