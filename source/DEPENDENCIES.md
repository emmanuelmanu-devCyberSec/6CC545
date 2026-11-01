# Week 5 dependency preparation

The direct runtime pins retain the versions in the earlier supplied assessment package: google.golang.org/grpc v1.75.1 and google.golang.org/protobuf v1.36.6. Generator pins: protoc-gen-go v1.36.6 and protoc-gen-go-grpc v1.5.1. These are compatibility pins, not a claim that they are the newest releases or a production security approval. Use the current supported Go toolchain approved by the module team. Record the selected protoc compiler version.

Run scripts/generate.sh or scripts/generate.ps1 from a network-enabled lab machine with protoc installed. This creates files.pb.go, files_grpc.pb.go and a genuine go.sum. The distributed scaffold deliberately contains no invented go.sum and no manually fabricated generated bindings. Missing generated symbols before this preparation are expected; they are not business-logic test failures.

Commit the .proto, generated .pb.go files, go.mod, go.sum, and version record. Do not commit bin/tools or a vendor/cache directory. go mod tidy may add indirect requirements; commit them rather than guessing transitive versions. Never disable checksum verification to make a dependency error disappear.

After successful preparation, go build ./... must work even before TODOs are implemented. go test ./... will then fail on genuine incomplete behaviours until you implement them.
