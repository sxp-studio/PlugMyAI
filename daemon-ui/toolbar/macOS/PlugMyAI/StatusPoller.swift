import Foundation

/// Polls the daemon's /v1/status endpoint on a timer and reports changes.
final class StatusPoller {
    private let url: URL
    private let interval: TimeInterval
    private let onChange: (DaemonStatus?) -> Void

    private var timer: Timer?
    private let session: URLSession

    init(port: Int = 21110, interval: TimeInterval = 5, onChange: @escaping (DaemonStatus?) -> Void) {
        self.url = URL(string: "http://localhost:\(port)/v1/status")!
        self.interval = interval
        self.onChange = onChange

        let config = URLSessionConfiguration.ephemeral
        config.timeoutIntervalForRequest = 3
        config.timeoutIntervalForResource = 3
        self.session = URLSession(configuration: config)
    }

    func start() {
        poll() // immediate first check
        timer = Timer.scheduledTimer(withTimeInterval: interval, repeats: true) { [weak self] _ in
            self?.poll()
        }
    }

    func stop() {
        timer?.invalidate()
        timer = nil
    }

    private func poll() {
        let request = URLRequest(url: url)
        session.dataTask(with: request) { [weak self] data, response, error in
            guard let self else { return }
            guard let data,
                  let http = response as? HTTPURLResponse,
                  http.statusCode == 200 else {
                DispatchQueue.main.async { self.onChange(nil) }
                return
            }
            let status = try? JSONDecoder().decode(DaemonStatus.self, from: data)
            DispatchQueue.main.async { self.onChange(status) }
        }.resume()
    }
}
