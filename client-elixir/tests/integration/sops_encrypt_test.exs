defmodule Signet.SopsEncryptIntegrationTest do
  # The real sops binary must decrypt what Signet.SopsEncrypt produces, as the
  # Go client's TestEncryptForSecret_RealSopsCanDecrypt checks: the document
  # is for signet's own sops-compatible decrypt path, not for this library's
  # test-support code. Runs in the toolchain image, where the Dockerfile
  # installs sops; scripts/integration-client-elixir.sh includes the tag.
  use ExUnit.Case, async: false

  alias Signet.AgeTestSupport
  alias Signet.SopsEncrypt

  @moduletag :integration

  setup do
    sops =
      System.find_executable("sops") ||
        flunk("sops binary not found on PATH; run this tier in the toolchain image")

    dir = Path.join(System.tmp_dir!(), "signet-sops-#{System.unique_integer([:positive])}")
    File.mkdir_p!(dir)
    on_exit(fn -> File.rm_rf!(dir) end)
    {:ok, sops: sops, dir: dir}
  end

  describe "sops --decrypt" do
    test "recovers the value with the identity it was encrypted to", ctx do
      identity = AgeTestSupport.generate_identity()
      path = write_document(ctx.dir, identity.recipient, "hello-world")

      assert {"hello-world", 0} = decrypt(ctx.sops, path, identity.identity)
    end

    test "recovers a value with newlines and non-ASCII bytes", ctx do
      identity = AgeTestSupport.generate_identity()
      value = "line one\nline two: still one value\n  ünïcödé ✓\n"
      path = write_document(ctx.dir, identity.recipient, value)

      assert {^value, 0} = decrypt(ctx.sops, path, identity.identity)
    end

    test "refuses an identity the document was not encrypted to", ctx do
      encrypted_to = AgeTestSupport.generate_identity()
      wrong = AgeTestSupport.generate_identity()
      path = write_document(ctx.dir, encrypted_to.recipient, "hello-world")

      assert {output, status} = decrypt(ctx.sops, path, wrong.identity)
      assert status != 0, "expected sops to refuse the wrong identity, got: #{output}"
    end
  end

  defp write_document(dir, recipient, value) do
    assert {:ok, yaml} = SopsEncrypt.encrypt_for_secret(recipient, value)
    path = Path.join(dir, "secret.yaml")
    File.write!(path, yaml)
    path
  end

  # --extract prints the leaf's raw value, so the assertion does not depend
  # on how sops's YAML output quotes it.
  defp decrypt(sops, path, identity) do
    System.cmd(sops, ["--decrypt", "--extract", ~s(["value"]), path],
      env: [{"SOPS_AGE_KEY", identity}],
      stderr_to_stdout: true
    )
  end
end
