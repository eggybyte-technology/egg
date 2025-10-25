# Connect Service Tester

A simple testing tool for Connect RPC services built with the egg framework.

## Features

- Human-readable console logging
- Tests Connect RPC endpoints
- Clear test output with timing
- Built using egg/logx

## Usage

First, start the service you want to test:

```bash
cd ../minimal-connect-service
go run main.go
```

Then run the tester:

```bash
# Build
go build -o connect-tester

# Run tests
./connect-tester http://localhost:8080
```

## Example Output

```
INFO    2024-10-25 15:30:00  connect service tester
        url: http://localhost:8080
INFO    2024-10-25 15:30:00  testing SayHello endpoint
INFO    2024-10-25 15:30:00  SayHello success
        duration: 25ms
        message: Hello, Tester!
INFO    2024-10-25 15:30:00  testing SayHelloStream endpoint
INFO    2024-10-25 15:30:00  SayHelloStream success
        duration: 305ms
        messages: 3
INFO    2024-10-25 15:30:00  all tests passed
```

## Implementation Highlights

- Uses `logx.FormatConsole` for human-readable output
- Clean error handling with context
- Simple test structure for easy extension

## Adding More Tests

To add tests for other services:

1. Import the service protobuf client
2. Add test functions like `testSayHello`
3. Call them from `runTests`

## License

This example is part of the EggyByte egg framework and is licensed under the MIT License.

