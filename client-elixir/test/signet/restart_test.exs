defmodule Signet.RestartTest do
  use ExUnit.Case, async: true

  alias Signet.FakeRestartStreams, as: Fake
  alias Signet.Restart

  # A watch stream the test feeds one item at a time with `feed/2`; it reports
  # the reader process consuming it.
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

  defp feed(reader), do: send(reader, {:item, {:ok, Fake.changed()}})

  # Runs wait_for_restart in a task, with the test process as the fake's
  # recipient and the lock's owner, and returns the task and the watch reader.
  defp wait(debounce_ms, opts \\ []) do
    channel = Fake.start([watch: [{:ok, feedable(self())}]], self())
    owner = self()
    opts = [streams: Fake, owner: owner] ++ opts

    task =
      Task.async(fn ->
        Restart.wait_for_restart(channel, "ns", "svc", 4_000, debounce_ms, opts)
      end)

    assert_receive {:reader, reader}
    {channel, task, reader}
  end

  defp expires, do: DateTime.add(DateTime.utc_now(), 30)

  test "wait_for_restart acquires right after the first change with no debounce" do
    {channel, task, reader} = wait(0)
    refute_receive {:lock_opened, _, _}, 100

    feed(reader)
    assert_receive {:lock_opened, ^channel, stream}
    Fake.push(stream, {:ok, Fake.acquired("tok", expires())})

    assert {:ok, lock} = Task.await(task)
    assert Restart.token(lock) == "tok"
    assert %DateTime{} = Restart.expires_at(lock)
  end

  test "wait_for_restart debounces changes that keep arriving" do
    {channel, task, reader} = wait(300)
    ref = Process.monitor(reader)

    feed(reader)
    Process.sleep(150)
    feed(reader)
    Process.sleep(150)
    feed(reader)
    refute_receive {:lock_opened, _, _}, 100

    assert_receive {:lock_opened, ^channel, stream}, 500
    assert_receive {:DOWN, ^ref, :process, ^reader, :shutdown}
    Fake.push(stream, {:ok, Fake.acquired("tok", expires())})
    assert {:ok, _lock} = Task.await(task)
  end

  test "wait_for_restart stops the watcher before acquiring" do
    {channel, task, reader} = wait(0)
    ref = Process.monitor(reader)

    feed(reader)
    assert_receive {:lock_opened, ^channel, stream}
    assert_receive {:DOWN, ^ref, :process, ^reader, :shutdown}

    Fake.push(stream, {:ok, Fake.acquired("tok", expires())})
    assert {:ok, _lock} = Task.await(task)
  end

  test "wait_for_restart returns a lock failure and leaves the watcher stopped" do
    {channel, task, reader} = wait(0)
    ref = Process.monitor(reader)

    feed(reader)
    assert_receive {:lock_opened, ^channel, stream}
    Fake.push(stream, {:error, :boom})

    assert {:error, :boom} = Task.await(task)
    assert_receive {:DOWN, ^ref, :process, ^reader, :shutdown}
  end

  test "wait_for_restart rejects a bad ttl or debounce before watching" do
    channel = Fake.start([], self())

    assert {:error, reason} = Restart.wait_for_restart(channel, "ns", "svc", 0, 0, streams: Fake)
    assert reason =~ "ttl must be > 0"

    assert {:error, reason} =
             Restart.wait_for_restart(channel, "ns", "svc", 4_000, -1, streams: Fake)

    assert reason =~ "debounce must be >= 0"

    refute_receive {:watch, _, _}
  end

  test "watch_bundle, next_change, acknowledge, and stop_watch round trip" do
    channel = Fake.start([watch: [{:ok, feedable(self())}]], self())
    {:ok, watcher} = Restart.watch_bundle(channel, "ns", "svc", streams: Fake)
    assert_receive {:reader, reader}

    assert Restart.next_change(watcher, 50) == :timeout
    feed(reader)
    assert Restart.next_change(watcher) == :ok

    feed(reader)
    assert_receive {:bundle_changed, ^watcher}
    assert :ok = Restart.acknowledge(watcher)

    assert :ok = Restart.stop_watch(watcher)
    refute Process.alive?(watcher)
  end

  test "acquire_lock and release round trip" do
    channel = Fake.start([], self())
    owner = self()

    task =
      Task.async(fn ->
        Restart.acquire_lock(channel, "ns", "svc", 4_000, streams: Fake, owner: owner)
      end)

    assert_receive {:lock_opened, ^channel, stream}
    Fake.push(stream, {:ok, Fake.acquired("tok-1", expires())})

    assert {:ok, lock} = Task.await(task)
    assert Restart.token(lock) == "tok-1"
    assert :ok = Restart.release(lock)
    assert :ok = Restart.release(lock)
    assert Fake.closed?(stream)
  end
end
