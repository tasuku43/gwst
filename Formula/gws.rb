class Gws < Formula
  desc "Git Workspaces for Human + Agentic Development"
  homepage "https://github.com/tasuku43/gws"
  license "MIT"

  version "0.0.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/tasuku43/gws/releases/download/v0.0.0/gws_v0.0.0_macos_arm64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    else
      url "https://github.com/tasuku43/gws/releases/download/v0.0.0/gws_v0.0.0_macos_x64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/tasuku43/gws/releases/download/v0.0.0/gws_v0.0.0_linux_arm64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    else
      url "https://github.com/tasuku43/gws/releases/download/v0.0.0/gws_v0.0.0_linux_x64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  def install
    bin.install "gws"
  end

  test do
    system "#{bin}/gws", "--version"
  end
end

