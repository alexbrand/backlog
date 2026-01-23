# typed: false
# frozen_string_literal: true

# Homebrew formula for backlog CLI
# To install: brew install alexbrand/tap/backlog
# Or: brew tap alexbrand/tap && brew install backlog
class Backlog < Formula
  desc "CLI tool for managing tasks across multiple issue tracking backends"
  homepage "https://github.com/alexbrand/backlog"
  version "0.1.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/alexbrand/backlog/releases/download/v#{version}/backlog-darwin-arm64"
      sha256 "PLACEHOLDER_DARWIN_ARM64_SHA256"

      def install
        bin.install "backlog-darwin-arm64" => "backlog"
      end
    end

    on_intel do
      url "https://github.com/alexbrand/backlog/releases/download/v#{version}/backlog-darwin-amd64"
      sha256 "PLACEHOLDER_DARWIN_AMD64_SHA256"

      def install
        bin.install "backlog-darwin-amd64" => "backlog"
      end
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/alexbrand/backlog/releases/download/v#{version}/backlog-linux-arm64"
      sha256 "PLACEHOLDER_LINUX_ARM64_SHA256"

      def install
        bin.install "backlog-linux-arm64" => "backlog"
      end
    end

    on_intel do
      url "https://github.com/alexbrand/backlog/releases/download/v#{version}/backlog-linux-amd64"
      sha256 "PLACEHOLDER_LINUX_AMD64_SHA256"

      def install
        bin.install "backlog-linux-amd64" => "backlog"
      end
    end
  end

  test do
    assert_match "backlog version", shell_output("#{bin}/backlog version")
  end
end
