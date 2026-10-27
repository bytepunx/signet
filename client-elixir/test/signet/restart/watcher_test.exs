defmodule Signet.Restart.WatcherTest do
  use ExUnit.Case, async: true

  alias Signet.FakeRestartStreams, as: Fake
  alias Signet.Restart.Watcher
  alias Signet.V1.WatchServiceBundleRequest

  defp start(script) do
    channel = Fake.start([watch: script], self())
    test = self()
    sleep = fn ms -> send(test, {:slept, ms}) end
    {:ok, watcher} = Watcher.start_link(channel, "ns", "svc", streams: Fake, sleep: sleep)
    {channel, watcher}
  end

  defp slept do
    receive do
      {:slept, ms} -> [ms | slept()]
    after
      0 -> []
    end
  end

  # A stream that yields `items` and then stays open and silent.
  defp then_silent(items) do
    Stream.concat(items, Stream.repeatedly(fn -> Process.sleep(:infinity) end))
  end

  # A stream the test feeds one item at a time with `feed/2`; it reports the
  # reader process consuming it.
  defp feedable(test) do
    Stream.resource(
      fn -> send(test, {:reader, self()}) end,
      fn acc ->
        receive do
          {:item, item} -> {[item], acc}
        end
      end,
      fn _ -> :ok end
    )
  end

  defp feed(reader, item), do: send(reader, {:item, item})

  defp changed, do: {:ok, Fake.changed()}

  test "watches the namespace and service it was started for" do
    {channel, _watcher} = start([])
    assert_receive {:watch, ^channel, %WatchServiceBundleRequest{namespace: "ns", service: "svc"}}
  end

  test "delivers one message per burst, and the next only once acknowledged" do
    {_channel, watcher} = start([{:ok, feedable(self())}])
    assert_receive {:reader, reader}

    for _ <- 1..3, do: feed(reader, changed())
    assert_receive {:bundle_changed, ^watcher}
    refute_receive {:bundle_changed, _}, 100

    feed(reader, changed())
    refute_receive {:bundle_changed, _}, 100

    :ok = Watcher.acknowledge(watcher)
    feed(reader, changed())
    assert_receive {:bundle_changed, ^watcher}
  end

  test "next_change receives and acknowledges, or times out" do
    {_channel, watcher} = start([{:ok, feedable(self())}])
    assert_receive {:reader, reader}

    assert Watcher.next_change(watcher, 50) == :timeout

    feed(reader, changed())
    assert Watcher.next_change(watcher) == :ok

    feed(reader, changed())
    assert Watcher.next_change(watcher, 1_000) == :ok
  end

  test "reopens after a failed open with backoff 1s then 2s" do
    {_channel, watcher} =
      start([{:error, :boom}, {:error, :boom}, {:ok, then_silent([changed()])}])

    assert_receive {:bundle_changed, ^watcher}
    assert slept() == [1_000, 2_000]
  end

  test "caps the backoff at 30s" do
    errors = List.duplicate({:error, :boom}, 7)
    {_channel, watcher} = start(errors ++ [{:ok, then_silent([changed()])}])
    assert_receive {:bundle_changed, ^watcher}
    assert slept() == [1_000, 2_000, 4_000, 8_000, 16_000, 30_000, 30_000]
  end

  test "resets the backoff after any event, as the Go client does" do
    {channel, watcher} =
      start([
        {:error, :boom},
        {:ok, [changed()]},
        {:error, :boom},
        {:ok, then_silent([changed()])}
      ])

    for _ <- 1..4, do: assert_receive({:watch, ^channel, _})
    assert_receive {:bundle_changed, ^watcher}
    assert slept() == [1_000, 1_000, 2_000]
  end

  test "treats a stream error like an end: sleeps and reopens" do
    {channel, _watcher} = start([{:ok, [{:error, :broken}]}, {:ok, then_silent([])}])
    assert_receive {:watch, ^channel, _}
    assert_receive {:watch, ^channel, _}
    assert slept() == [1_000]
  end

  test "stop ends the watcher and its reader" do
    {_channel, watcher} = start([{:ok, feedable(self())}])
    assert_receive {:reader, reader}
    ref = Process.monitor(reader)

    :ok = Watcher.stop(watcher)

    assert_receive {:DOWN, ^ref, :process, ^reader, :shutdown}
    refute Process.alive?(watcher)
  end
end
