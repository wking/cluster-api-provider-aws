load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["deep_equal.go"],
    importmap = "sigs.k8s.io/cluster-api-provider-aws/vendor/k8s.io/apimachinery/third_party/forked/golang/reflect",
    importpath = "k8s.io/apimachinery/third_party/forked/golang/reflect",
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_test",
    srcs = ["deep_equal_test.go"],
    embed = [":go_default_library"],
)
