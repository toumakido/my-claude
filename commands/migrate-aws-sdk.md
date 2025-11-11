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
   - Error handling
4. Execute automatic migration for each file:
   - Update import statements
   - Replace session initialization with config.LoadDefaultConfig
   - Update service client creation
   - Add context parameters to API calls
   - Preserve existing logic and error handling
5. Update go.mod dependencies:
   - Add v2 dependencies: `go get github.com/aws/aws-sdk-go-v2/config`
   - Add required service packages
   - Run `go mod tidy`
6. Verify compilation: `go build ./...`
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

## Notes

- Backup or commit before running migration
- Test thoroughly after migration - some APIs have behavioral changes
- Check AWS SDK Go v2 migration guide for service-specific changes
- Update unit tests to match new patterns
- Consider using existing context.Context from caller functions instead of context.TODO()
