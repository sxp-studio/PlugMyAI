import Foundation

/// Codable model matching the Go daemon's GET /v1/status response.
struct DaemonStatus: Codable {
    let status: String
    let version: String
    let uptimeS: Int
    let port: Int
    let providers: [String]

    enum CodingKeys: String, CodingKey {
        case status, version, port, providers
        case uptimeS = "uptime_s"
    }

    var isRunning: Bool { status == "ok" }
    var providerCount: Int { providers.count }
}
