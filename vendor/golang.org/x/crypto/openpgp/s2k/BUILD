load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["s2k.go"],
    importmap = "sigs.k8s.io/cluster-api-provider-aws/vendor/golang.org/x/crypto/openpgp/s2k",
    importpath = "golang.org/x/crypto/openpgp/s2k",
    visibility = ["//visibility:public"],
    deps = ["//vendor/golang.org/x/crypto/openpgp/errors:go_default_library"],
)

go_test(
    name = "go_default_test",
    srcs = ["s2k_test.go"],
    embed = [":go_default_library"],
    deps = ["//vendor/golang.org/x/crypto/ripemd160:go_default_library"],
)
