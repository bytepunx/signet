defmodule Signet.FakeWorkloadAPI do
  @moduledoc """
  A `Signet.Spiffe.WorkloadAPI` whose answers a test scripts in advance, one
  per `fetch_x509_svid/1` call, and that reports each call to the test.

  The script is a list of `{:ok, stream}` or `{:error, reason}` values held in
  an Agent registered under the socket path, so the test and the source's
  reader process share it.
  """

  @behaviour Signet.Spiffe.WorkloadAPI

  def start(socket_path, script, test_pid) do
    {:ok, _} =
      Agent.start_link(fn -> %{script: script, test: test_pid} end, name: name(socket_path))

    :ok
  end

  @impl true
  def fetch_x509_svid(socket_path) do
    Agent.get_and_update(name(socket_path), fn
      %{script: [answer | rest], test: test} = state ->
        send(test, {:fetch, socket_path})
        {answer, %{state | script: rest}}

      # Script exhausted: the stream stays open and silent, as a real agent's
      # does between rotations.
      %{test: test} = state ->
        send(test, {:fetch, socket_path})
        {{:ok, Stream.repeatedly(fn -> Process.sleep(:infinity) end)}, state}
    end)
  end

  def no_identity_issued do
    GRPC.RPCError.exception(GRPC.Status.permission_denied(), "no identity issued")
  end

  defp name(socket_path), do: {:global, {__MODULE__, socket_path}}
end
