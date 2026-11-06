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

# dotnet's first-run setup crashes (NuGet-Migrations IOException) when the uid
# has no /etc/passwd entry, as on CI (uid 1001). Name the caller in a passwd
# and group file mounted read-only over the image's, so --user can stay.
uid="$(id -u)"
gid="$(id -g)"
ids=".cache/dotnet-ids"
mkdir -p "${ids}"
printf 'root:x:0:0:root:/root:/bin/sh\ncaller:x:%s:%s::/cache:/bin/sh\n' "${uid}" "${gid}" > "${ids}/passwd"
printf 'root:x:0:\ncaller:x:%s:\n' "${gid}" > "${ids}/group"

run() {
  docker run --rm \
    --user "${uid}:${gid}" \
    -v "$(pwd)/${ids}/passwd:/etc/passwd:ro" \
    -v "$(pwd)/${ids}/group:/etc/group:ro" \
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
