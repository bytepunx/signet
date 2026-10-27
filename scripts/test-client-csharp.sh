#!/usr/bin/env sh
# Behavior tests for the C# client, run inside Docker so nothing is installed
# on the host. Generates the stubs into client-csharp/gen (gitignored) with
# scripts/buf-generate.sh, then with client-csharp/ mounted at /app runs
# dotnet test on the solution. bin/ and obj/ stay on the host, gitignored,
# owned by the caller; the NuGet cache lives in .cache/dotnet.
#
#   scripts/test-client-csharp.sh                       # dotnet test SignetClient.sln
#   scripts/test-client-csharp.sh dotnet build          # any other command
set -eu
cd "$(dirname "$0")/.."

image="mcr.microsoft.com/dotnet/sdk:8.0"
cache=".cache/dotnet"
mkdir -p "${cache}"

echo "test-client-csharp.sh: generating stubs"
scripts/buf-generate.sh client-csharp

run() {
  docker run --rm \
    --user "$(id -u):$(id -g)" \
    -e HOME=/cache \
    -e DOTNET_CLI_HOME=/cache \
    -e NUGET_PACKAGES=/cache/nuget \
    -e DOTNET_NOLOGO=1 \
    -e DOTNET_CLI_TELEMETRY_OPTOUT=1 \
    -v "$(pwd)/client-csharp:/app" \
    -v "$(pwd)/${cache}:/cache" \
    -w /app \
    "${image}" "$@"
}

if [ "$#" -gt 0 ]; then
  echo "test-client-csharp.sh: running $*"
  run "$@"
else
  echo "test-client-csharp.sh: testing"
  run dotnet test SignetClient.sln
fi
