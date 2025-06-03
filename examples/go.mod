module example.com/test-project

go 1.21

require (
	google.golang.org/grpc v1.60.1
	google.golang.org/protobuf v1.32.0
)

// Protobuf libraries
replace local-product-api v0.0.0 => github.com/example/product-api v0.12.0
replace local-user-api v0.0.0 => github.com/example/user-api v0.8.5
replace local-common-protos v0.0.0 => github.com/example/common-protos v0.15.2 