class Gws < Formula
  desc "Git Workspaces for Human + Agentic Development"
  homepage "https://github.com/tasuku43/gws"
  license "MIT"

  version "0.1.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/tasuku43/gws/releases/download/v0.1.0/gws_v0.1.0_macos_arm64.tar.gz"
      sha256 "0b6b0149ab9050bcc2f8cbb9b5cbbb3eac4a6ebcea43a1f3036b636c92fcbb72"
    else
      url "https://github.com/tasuku43/gws/releases/download/v0.1.0/gws_v0.1.0_macos_x64.tar.gz"
      sha256 "4200679e3e1ffe4440656858aaef5cec84e0d08a994bb60c2d06a24cfa9409f2"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/tasuku43/gws/releases/download/v0.1.0/gws_v0.1.0_linux_arm64.tar.gz"
      sha256 "d82367681c9fbc84e2245b72490d11db4763053f0306b323edebda44269a10ec"
    else
      url "https://github.com/tasuku43/gws/releases/download/v0.1.0/gws_v0.1.0_linux_x64.tar.gz"
      sha256 "ff8ee8a7e9d5d3db6da24d85f0fc60c454b870cd846dc0d8ed164b053c5e6c54"
    end
  end

  def install
    bin.install "gws"
  end

  test do
    system "#{bin}/gws", "--version"
  end
end
