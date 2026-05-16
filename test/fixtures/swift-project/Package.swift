// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "OrderApp",
    products: [
        .library(name: "OrderApp", targets: ["Domain", "Application", "Infrastructure"]),
    ],
    targets: [
        .target(name: "Domain"),
        .target(name: "Application", dependencies: ["Domain", "Infrastructure"]),
        .target(name: "Infrastructure", dependencies: ["Domain"]),
        .testTarget(name: "OrderTests", dependencies: ["Domain", "Application", "Infrastructure"]),
    ]
)
