load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "compression.go",
        "diskv.go",
        "index.go",
    ],
    importmap = "sigs.k8s.io/cluster-api-provider-aws/vendor/github.com/peterbourgon/diskv",
    importpath = "github.com/peterbourgon/diskv",
    visibility = ["//visibility:public"],
    deps = ["//vendor/github.com/google/btree:go_default_library"],
)

go_test(
    name = "go_default_test",
    srcs = [
        "basic_test.go",
        "compression_test.go",
        "import_test.go",
        "index_test.go",
        "issues_test.go",
        "keys_test.go",
        "speed_test.go",
        "stream_test.go",
    ],
    embed = [":go_default_library"],
)
