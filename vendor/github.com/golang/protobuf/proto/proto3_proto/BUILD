load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["proto3.pb.go"],
    importmap = "sigs.k8s.io/cluster-api-provider-aws/vendor/github.com/golang/protobuf/proto/proto3_proto",
    importpath = "github.com/golang/protobuf/proto/proto3_proto",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/github.com/golang/protobuf/proto:go_default_library",
        "//vendor/github.com/golang/protobuf/proto/test_proto:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes/any:go_default_library",
    ],
)
