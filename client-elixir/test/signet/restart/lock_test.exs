defmodule Signet.Restart.LockTest do
  use ExUnit.Case, async: true

  alias Signet.FakeRestartStreams, as: Fake
  alias Signet.FakeRestartStreams.LockStream
  alias Signet.Restart.Lock
  alias Signet.V1.AcquireRestartLockRequest

  # Starts an acquire in a task, so the test process stays free to drive the
  # fake stream, and returns the task and the fake stream it opened. The lock
  # reports loss to the test process, not to the task.
  defp acquire(ttl_ms, opts \\ []) do
    channel = Fake.start([], self())
    owner = self()
    opts = [streams: Fake, owner: owner] ++ opts
    task = Task.async(fn -> Lock.acquire(channel, "ns", "svc", ttl_ms, opts) end)
    assert_receive {:lock_opened, ^channel, stream}
    {task, stream}
  end

  defp acquired(ttl_ms, opts \\ []) do
    {task, stream} = acquire(ttl_ms, opts)
    expires = DateTime.add(DateTime.utc_now(), 30)
    Fake.push(stream, {:ok, Fake.acquired("tok-1", expires)})
    {:ok, lock} = Task.await(task)
    {lock, stream, expires}
  end

  defp eventually(check, tries \\ 50) do
    cond do
      check.() -> true
      tries == 0 -> false
      true -> Process.sleep(10) && eventually(check, tries - 1)
    end
  end

  test "rejects a non-positive ttl without opening a stream" do
    channel = Fake.start([], self())
    assert {:error, reason} = Lock.acquire(channel, "ns", "svc", 0, streams: Fake)
    assert reason =~ "ttl must be > 0"
    assert {:error, _} = Lock.acquire(channel, "ns", "svc", -1, streams: Fake)
    refute_receive {:lock_opened, _, _}
  end

  test "queues behind the holder and acquires, with one request sent before heartbeats" do
    {task, stream} = acquire(4_000)

    assert_receive {:lock_sent, ^stream,
                    %AcquireRestartLockRequest{
                      namespace: "ns",
                      service: "svc",
                      ttl_seconds: 4,
                      heartbeat: false
                    }}

    expires = DateTime.add(DateTime.utc_now(), 30)
    Fake.push(stream, {:ok, Fake.queue_position(2)})
    Fake.push(stream, {:ok, Fake.queue_position(1)})
    Fake.push(stream, {:ok, Fake.acquired("tok-1", expires)})

    assert {:ok, lock} = Task.await(task)
    assert Lock.token(lock) == "tok-1"
    assert DateTime.compare(Lock.expires_at(lock), expires) == :eq
    assert [%AcquireRestartLockRequest{heartbeat: false}] = Fake.sent(stream)
    refute Lock.released?(lock)
  end

  test "carries a sub-second ttl as one second on the wire" do
    {task, stream} = acquire(400)
    assert_receive {:lock_sent, ^stream, %AcquireRestartLockRequest{ttl_seconds: 1}}
    Fake.push(stream, {:error, :boom})
    assert {:error, :boom} = Task.await(task)
  end

  test "a stream error before ACQUIRED is returned and the stream closed" do
    {task, stream} = acquire(4_000)
    Fake.push(stream, {:error, :boom})
    assert {:error, :boom} = Task.await(task)
    assert Fake.closed?(stream)
  end

  test "the server closing the stream before ACQUIRED is an error" do
    {task, stream} = acquire(4_000)
    LockStream.close(stream)
    assert {:error, :closed} = Task.await(task)
  end

  test "gives up after the timeout and closes the stream" do
    {task, stream} = acquire(4_000, timeout: 50)
    Fake.push(stream, {:ok, Fake.queue_position(3)})
    assert {:error, :timeout} = Task.await(task)
    assert Fake.closed?(stream)
  end

  test "heartbeats at ttl/4 once acquired" do
    {_lock, stream, _expires} = acquired(400)
    assert_receive {:lock_sent, ^stream, %AcquireRestartLockRequest{heartbeat: true}}, 500
    assert_receive {:lock_sent, ^stream, %AcquireRestartLockRequest{heartbeat: true}}, 500
    assert [%{heartbeat: false} | heartbeats] = Fake.sent(stream)
    assert length(heartbeats) >= 2
    assert Enum.all?(heartbeats, & &1.heartbeat)
  end

  test "TTL_EXTENDED moves the expiry" do
    {lock, stream, first} = acquired(4_000)
    assert DateTime.compare(Lock.expires_at(lock), first) == :eq

    extended = DateTime.add(DateTime.utc_now(), 60)
    Fake.push(stream, {:ok, Fake.ttl_extended(extended)})
    assert eventually(fn -> DateTime.compare(Lock.expires_at(lock), extended) == :eq end)
  end

  test "release closes the stream, is idempotent, and keeps answering" do
    {lock, stream, _expires} = acquired(4_000)

    assert :ok = Lock.release(lock)
    assert Fake.closed?(stream)
    assert Lock.released?(lock)
    assert :ok = Lock.release(lock)
    assert Lock.token(lock) == "tok-1"
    assert Process.alive?(lock)
  end

  test "release stops the heartbeats" do
    {lock, stream, _expires} = acquired(400)
    :ok = Lock.release(lock)
    sent_at_release = length(Fake.sent(stream))
    Process.sleep(300)
    assert length(Fake.sent(stream)) == sent_at_release
  end

  test "reports loss once on an unexpected stream error" do
    {lock, stream, _expires} = acquired(400)
    Fake.push(stream, {:error, :lost})
    assert_receive {:lock_lost, ^lock, :lost}

    LockStream.close(stream)
    refute_receive {:lock_lost, _, _}, 300
  end

  test "does not report loss after a release" do
    {lock, stream, _expires} = acquired(4_000)
    :ok = Lock.release(lock)
    assert eventually(fn -> Fake.closed?(stream) end)
    refute_receive {:lock_lost, _, _}, 200
  end

  test "reports loss once when a heartbeat cannot be sent" do
    {lock, stream, _expires} = acquired(400)
    LockStream.close(stream)
    assert_receive {:lock_lost, ^lock, _reason}, 500
    refute_receive {:lock_lost, _, _}, 300
  end

  test "stop releases and ends the process" do
    {lock, stream, _expires} = acquired(4_000)
    assert :ok = Lock.stop(lock)
    assert Fake.closed?(stream)
    refute Process.alive?(lock)
  end
end
