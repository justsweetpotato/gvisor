load("//tools:defs.bzl", "go_library", "go_test")
load("//tools/go_generics:defs.bzl", "go_template_instance")

package(licenses = ["notice"])

go_template_instance(
    name = "linkaddrentry_list",
    out = "linkaddrentry_list.go",
    package = "stack",
    prefix = "linkAddrEntry",
    template = "//pkg/ilist:generic_list",
    types = {
        "Element": "*linkAddrEntry",
        "Linker": "*linkAddrEntry",
    },
)

go_library(
    name = "stack",
    srcs = [
        "icmp_rate_limit.go",
        "linkaddrcache.go",
        "linkaddrentry_list.go",
        "ndp.go",
        "nic.go",
        "registration.go",
        "route.go",
        "stack.go",
        "stack_global_state.go",
        "transport_demuxer.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/ilist",
        "//pkg/rand",
        "//pkg/sleep",
        "//pkg/sync",
        "//pkg/tcpip",
        "//pkg/tcpip/buffer",
        "//pkg/tcpip/hash/jenkins",
        "//pkg/tcpip/header",
        "//pkg/tcpip/iptables",
        "//pkg/tcpip/ports",
        "//pkg/tcpip/seqnum",
        "//pkg/waiter",
        "@org_golang_x_time//rate:go_default_library",
    ],
)

go_test(
    name = "stack_x_test",
    size = "medium",
    srcs = [
        "ndp_test.go",
        "stack_test.go",
        "transport_demuxer_test.go",
        "transport_test.go",
    ],
    deps = [
        ":stack",
        "//pkg/rand",
        "//pkg/tcpip",
        "//pkg/tcpip/buffer",
        "//pkg/tcpip/checker",
        "//pkg/tcpip/header",
        "//pkg/tcpip/iptables",
        "//pkg/tcpip/link/channel",
        "//pkg/tcpip/link/loopback",
        "//pkg/tcpip/network/ipv4",
        "//pkg/tcpip/network/ipv6",
        "//pkg/tcpip/transport/icmp",
        "//pkg/tcpip/transport/udp",
        "//pkg/waiter",
        "@com_github_google_go-cmp//cmp:go_default_library",
    ],
)

go_test(
    name = "stack_test",
    size = "small",
    srcs = [
        "linkaddrcache_test.go",
        "nic_test.go",
    ],
    library = ":stack",
    deps = [
        "//pkg/sleep",
        "//pkg/sync",
        "//pkg/tcpip",
        "//pkg/tcpip/buffer",
    ],
)
