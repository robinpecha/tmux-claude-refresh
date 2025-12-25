#!/bin/bash
set -euo pipefail

# Release script for autoclaude
# Usage: ./scripts/release.sh <version>
# Example: ./scripts/release.sh 0.0.2

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.0.2"
    exit 1
fi

TAG="v$VERSION"
REPO="henryaj/autoclaude"
TAP_REPO="henryaj/homebrew-tap"

echo "==> Releasing $TAG"

# Check for uncommitted changes
if [[ -n "$(git status --porcelain)" ]]; then
    echo "Error: Working directory has uncommitted changes"
    exit 1
fi

# Check if tag already exists
if git rev-parse "$TAG" >/dev/null 2>&1; then
    echo "Error: Tag $TAG already exists"
    exit 1
fi

# Create and push tag
echo "==> Creating tag $TAG"
git tag "$TAG"
git push origin "$TAG"

# Wait for release workflow to complete
echo "==> Waiting for release workflow..."
sleep 5

RUN_ID=$(gh run list -R "$REPO" -L 1 --json databaseId -q '.[0].databaseId')
echo "==> Watching workflow run $RUN_ID"
gh run watch "$RUN_ID" -R "$REPO" --exit-status || {
    echo "Error: Release workflow failed"
    echo "Check: https://github.com/$REPO/actions"
    exit 1
}

# Download checksums
echo "==> Downloading checksums"
CHECKSUMS=$(gh release download "$TAG" -R "$REPO" -p checksums.txt -O -)

# Extract checksums
DARWIN_AMD64_SHA=$(echo "$CHECKSUMS" | grep darwin_amd64 | awk '{print $1}')
DARWIN_ARM64_SHA=$(echo "$CHECKSUMS" | grep darwin_arm64 | awk '{print $1}')
LINUX_AMD64_SHA=$(echo "$CHECKSUMS" | grep linux_amd64 | awk '{print $1}')
LINUX_ARM64_SHA=$(echo "$CHECKSUMS" | grep linux_arm64 | awk '{print $1}')

# Clone tap repo
echo "==> Updating homebrew tap"
TMPDIR=$(mktemp -d)
git clone "git@github.com:$TAP_REPO.git" "$TMPDIR/tap" --quiet

# Generate formula
cat > "$TMPDIR/tap/autoclaude.rb" << EOF
class Autoclaude < Formula
  desc "Automatically resume Claude Code sessions after rate limits"
  homepage "https://github.com/$REPO"
  version "$VERSION"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/$REPO/releases/download/$TAG/autoclaude_${VERSION}_darwin_amd64.tar.gz"
      sha256 "$DARWIN_AMD64_SHA"
    end

    on_arm do
      url "https://github.com/$REPO/releases/download/$TAG/autoclaude_${VERSION}_darwin_arm64.tar.gz"
      sha256 "$DARWIN_ARM64_SHA"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/$REPO/releases/download/$TAG/autoclaude_${VERSION}_linux_amd64.tar.gz"
      sha256 "$LINUX_AMD64_SHA"
    end

    on_arm do
      url "https://github.com/$REPO/releases/download/$TAG/autoclaude_${VERSION}_linux_arm64.tar.gz"
      sha256 "$LINUX_ARM64_SHA"
    end
  end

  def install
    bin.install "autoclaude"
  end

  test do
    system "#{bin}/autoclaude", "-version"
  end
end
EOF

# Commit and push formula
cd "$TMPDIR/tap"
git add autoclaude.rb
git commit -m "Update autoclaude to $VERSION"
git push

# Cleanup
rm -rf "$TMPDIR"

echo "==> Release $TAG complete!"
echo "Install with: brew install henryaj/tap/autoclaude"
