defmodule Signet do
  @moduledoc """
  Connection helpers for talking to a signet server from Elixir.

  Two connection modes mirror the two listeners signet exposes, and
  `Signet.Restart` coordinates restarts over the workload connection:

    * `Signet.Workload.connect/3` connects to the workload-facing
      `SecretsService` over SPIFFE mTLS, with an X.509-SVID fetched from the
      SPIRE Workload API, the same credential mechanism the signet server
      itself uses to authenticate callers.
    * `Signet.Admin.connect/2` connects to the operator-facing `AdminService`
      and `GitOpsService` with a bearer token over TLS, or plaintext for
      loopback addresses, mirroring signet's own `signet` CLI.
    * `Signet.Restart.wait_for_restart/6` waits for a bundle change and then
      holds signet's restart lock, so that at most one replica restarts at a
      time, mirroring the Go client's `WaitForRestart`.

  The connections return a `GRPC.Channel` to pass to the generated stubs:
  `Signet.V1.SecretsService.Stub`, `Admin.V1.AdminService.Stub`, and
  `Admin.V1.GitOpsService.Stub`.
  """
end
