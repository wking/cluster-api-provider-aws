load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "merge.go",
        "rangetable.go",
        "tables10.0.0.go",
        "tables9.0.0.go",
    ],
    importmap = "sigs.k8s.io/cluster-api-provider-aws/vendor/golang.org/x/text/unicode/rangetable",
    importpath = "golang.org/x/text/unicode/rangetable",
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_test",
    srcs = [
        "merge_test.go",
        "rangetable_test.go",
    ],
    embed = [":go_default_library"],
)
