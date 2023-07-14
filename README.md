## Start
```shell
go get google.golang.org/grpc/cmd/protoc-gen-go-grpc
go get google.golang.org/protobuf/cmd/protoc-gen-go

go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
go install google.golang.org/protobuf/cmd/protoc-gen-go
```

## Generate
```shell
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/service.proto
```

## Reflection
```go
package pkg

import "google.golang.org/grpc/reflection"

func foo() {
    reflection.Register(server)
}
```

```shell
grpcurl -plaintext localhost:10116 list
```
```
grpc.reflection.v1alpha.ServerReflection
signservice.SignService
```

```shell
grpcurl -plaintext localhost:10116 describe
```
```
grpc.reflection.v1alpha.ServerReflection is a service:
service ServerReflection {
  rpc ServerReflectionInfo ( stream .grpc.reflection.v1alpha.ServerReflectionRequest ) returns ( stream .grpc.reflection.v1alpha.ServerReflectionResponse );
}
signservice.SignService is a service:
service SignService {
  rpc Sign ( .signservice.Document ) returns ( .signservice.DocSign );
  rpc SignStream ( stream .signservice.Document ) returns ( stream .signservice.DocSign );
  rpc Verify ( .signservice.VerifyRequest ) returns ( .signservice.VerifyResponse );
  rpc VerifyStream ( stream .signservice.VerifyRequest ) returns ( stream .signservice.VerifyResponse );
}
```

## Sign
Data must be base64 encoded.
```shell
grpcurl -plaintext -format json -d '{"data": "YXNkYXNkYXNkYXNkYXNk"}' localhost:10116 signservice.SignService.Sign
```

Possible output
```
{
  "sign": "REGt4dNPZYLaQEXXA/PKTxwFaMhIhTkfsvxyOMQVY0oqmFd7f/XBFG6aX3KvVkgOEh7UsndqZv9csdgDn6zqDg=="
}
```

## Verify
```shell
grpcurl -plaintext -format json -d '{"data": "YXNkYXNkYXNkYXNkYXNk"}' localhost:10116 signservice.SignService.Sign
```
Possible output
```
{
  "sign": "qf5+VF/juwmFQRMLedRzljnadrKeOQXUN0/SOBSQ3Gb79pS1/uOzJPhLNU1i0gEtdU3DLSA1SsfLIPLuhDanBg=="
}
```

```shell
grpcurl -plaintext -format json -d \
'{"doc":{"data": "YXNkYXNkYXNkYXNkYXNk"}, "sign":{"sign": "qf5+VF/juwmFQRMLedRzljnadrKeOQXUN0/SOBSQ3Gb79pS1/uOzJPhLNU1i0gEtdU3DLSA1SsfLIPLuhDanBg=="}}' \
localhost:10116 signservice.SignService.Verify
```

Possible output
```
{
  "isOk": true
}
```

Bad signature
```shell
grpcurl -emit-defaults -plaintext -format json -d \
'{"doc":{"data": "YXNkYXNkYXNkYXNkYXNk"}, "sign":{"sign": "qf5+VF/juwmFQRMLedRzljnadrKeOQXUN0/SOBSQ3Gb79pS1/uOzJPhLNU1i0gEtdU3DLSA1SsfLIPLuhDanBg=="}}' \
localhost:10116 signservice.SignService.Verify
```

```
{
  "isOk": false
}
```

## Benchmarks

| Bench name                                                      | Loop count |    ns/op |
|-----------------------------------------------------------------|-----------:|---------:|
| BenchmarkGrpcDocSignServer_Sign/size_17-16                      |      29068 |    40537 |
| BenchmarkGrpcDocSignServer_Sign/size_1024-16                    |      28004 |    42641 |
| BenchmarkGrpcDocSignServer_Sign/size_1024*1024-16               |        637 |  1861758 |
| BenchmarkGrpcDocSignServer_Sign/size_nil-16                     |      30822 |    39082 |
| BenchmarkGrpcDocSignServer_SignUnary-16                         |        596 |  1992559 |
| BenchmarkGrpcDocSignServer_SignBatch/size_0-16                  |      79906 |    15165 |
| BenchmarkGrpcDocSignServer_SignBatch/size_17-16                 |      29290 |    41040 |
| BenchmarkGrpcDocSignServer_SignBatch/size_6*17-16               |       8178 |   147086 |
| BenchmarkGrpcDocSignServer_SignBatch/size_5*1024+17-16          |       7665 |   159833 |
| BenchmarkGrpcDocSignServer_SignBatch/size_17+1024+1024*1024-16  |        621 |  1928025 |
| BenchmarkGrpcDocSignServer_SignStream/size_none-16              |  964539764 |    1.243 |
| BenchmarkGrpcDocSignServer_SignStream/size_17-16                |      41540 |    28521 |
| BenchmarkGrpcDocSignServer_SignStream/size_4*17-16              |      10000 |   102939 |
| BenchmarkGrpcDocSignServer_SignStream/size_3*1024+17-16         |      10000 |   109679 |
| BenchmarkGrpcDocSignServer_SignStream/size_20*1024*1024-16      |         31 | 35026141 |
| BenchmarkGrpcDocSignServer_SignStream/size_1024*1024+1024+17-16 |        681 |  1749070 |

Platform:
 - goos: darwin
 - goarch: arm64
 - cpu: Apple M1 Max

In bench.txt you can find raw results of a single run of the benchmark.

## REST API
```shell
go install \
    github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
    github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
    google.golang.org/protobuf/cmd/protoc-gen-go \
    google.golang.org/grpc/cmd/protoc-gen-go-grpc
```

```shell
protoc -I . --grpc-gateway_out . \
    --grpc-gateway_opt logtostderr=true \
    --grpc-gateway_opt paths=source_relative \
    --grpc-gateway_opt generate_unbound_methods=true \
    proto/service.proto
```

```shell
curl -H 'Authorization: jwt fgdfgdfgdgfdgfdgfdgfd' --data '{"data": "YXNkYXNkYXNkYXNkYXNk"}' localhost:8080/signservice.SignService/Sign                                                                                                                  
```
```
{"sign":"Jem/OJYUH6b0OseqsFCQA9zAhfBEPWpjUGMgO1BoQw2NkNj5Nb2MY6UkF9xPVNom7bb0E0fik9fEeNl/N4n7Bw=="}%
```

```shell
grpcurl -plaintext -H 'Authorization: jwt fgdfgdfgdgfdgfdgfdgfd' -format json -d '{"data": "YXNkYXNkYXNkYXNkYXNk"}' localhost:10116 signservice.SignService.Sign
```
```
{
  "sign": "Jem/OJYUH6b0OseqsFCQA9zAhfBEPWpjUGMgO1BoQw2NkNj5Nb2MY6UkF9xPVNom7bb0E0fik9fEeNl/N4n7Bw=="
}
```

## Helpful Links
 - https://learn.microsoft.com/en-us/aspnet/core/grpc/performance
 - https://github.com/grpc-ecosystem/go-grpc-middleware
 - https://github.com/grpc-ecosystem/grpc-gateway
 - https://blog.logrocket.com/guide-to-grpc-gateway/
 - https://protobuf.dev/programming-guides/encoding/
 - https://protobuf.dev/programming-guides/dos-donts/
 - https://learn.microsoft.com/en-us/dotnet/architecture/grpc-for-wcf-developers/streaming-versus-repeated