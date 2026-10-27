defmodule Signet.AgeTest do
  use ExUnit.Case, async: true

  alias Signet.Age
  alias Signet.Age.Bech32
  alias Signet.AgeTestSupport, as: Support

  @begin "-----BEGIN AGE ENCRYPTED FILE-----"
  @end_marker "-----END AGE ENCRYPTED FILE-----"

  describe "encrypt/2" do
    test "round-trips a 32-byte data key through the test decryptor" do
      identity = Support.generate_identity()
      data_key = :crypto.strong_rand_bytes(32)

      assert {:ok, armored} = Age.encrypt(identity.recipient, data_key)
      assert {:ok, ^data_key} = Support.decrypt(armored, identity.private)
    end

    test "round-trips an empty plaintext as one empty final chunk" do
      identity = Support.generate_identity()

      assert {:ok, armored} = Age.encrypt(identity.recipient, "")
      assert {:ok, ""} = Support.decrypt(armored, identity.private)
    end

    test "round-trips a plaintext larger than one 64 KiB chunk" do
      identity = Support.generate_identity()
      plaintext = :crypto.strong_rand_bytes(65_536 + 1000)

      assert {:ok, armored} = Age.encrypt(identity.recipient, plaintext)
      assert {:ok, ^plaintext} = Support.decrypt(armored, identity.private)
    end

    test "round-trips a plaintext of exactly one chunk" do
      identity = Support.generate_identity()
      plaintext = :crypto.strong_rand_bytes(65_536)

      assert {:ok, armored} = Age.encrypt(identity.recipient, plaintext)
      assert {:ok, ^plaintext} = Support.decrypt(armored, identity.private)
    end

    test "accepts an uppercase recipient" do
      identity = Support.generate_identity()

      assert {:ok, armored} = Age.encrypt(String.upcase(identity.recipient), "secret")
      assert {:ok, "secret"} = Support.decrypt(armored, identity.private)
    end

    test "armors as a base64 age v1 file between the markers, 64 columns wide" do
      identity = Support.generate_identity()
      {:ok, armored} = Age.encrypt(identity.recipient, "secret")

      assert String.starts_with?(armored, @begin <> "\n")
      assert String.ends_with?(armored, "\n" <> @end_marker <> "\n")

      lines =
        armored
        |> String.trim_trailing("\n")
        |> String.split("\n")
        |> Enum.drop(1)
        |> Enum.drop(-1)

      assert Enum.all?(lines, &(String.length(&1) <= 64))
      assert {:ok, file} = lines |> Enum.join() |> Base.decode64()
      assert String.starts_with?(file, "age-encryption.org/v1\n-> X25519 ")
    end

    test "a different identity cannot decrypt" do
      identity = Support.generate_identity()
      other = Support.generate_identity()
      {:ok, armored} = Age.encrypt(identity.recipient, "secret")

      assert {:error, :wrong_identity} = Support.decrypt(armored, other.private)
    end

    test "rejects an empty recipient" do
      assert {:error, reason} = Age.encrypt("", "secret")
      assert reason =~ "must not be empty"
    end

    test "rejects a recipient that is not bech32" do
      assert {:error, reason} = Age.encrypt("not-a-key", "secret")
      assert reason =~ "not a bech32 string"
    end

    test "rejects a recipient of another type" do
      recipient = Bech32.encode("foo", :crypto.strong_rand_bytes(32))

      assert {:error, reason} = Age.encrypt(recipient, "secret")
      assert reason =~ ~s(unknown recipient type "foo")
    end

    test "rejects a recipient whose key is not 32 bytes" do
      recipient = Bech32.encode("age", :crypto.strong_rand_bytes(16))

      assert {:error, reason} = Age.encrypt(recipient, "secret")
      assert reason =~ "16 bytes, want 32"
    end

    test "rejects a recipient with a bad checksum" do
      %{recipient: recipient} = Support.generate_identity()
      {head, last} = String.split_at(recipient, -1)
      damaged = head <> if(last == "q", do: "p", else: "q")

      assert {:error, reason} = Age.encrypt(damaged, "secret")
      assert reason =~ "not a bech32 string"
    end
  end

  describe "parse_recipient/1" do
    test "returns the public key bytes" do
      identity = Support.generate_identity()
      assert {:ok, key} = Age.parse_recipient(identity.recipient)
      assert key == identity.public
    end
  end

  describe "Bech32" do
    test "round-trips bytes under a human-readable part" do
      bytes = :crypto.strong_rand_bytes(32)
      encoded = Bech32.encode("age", bytes)

      assert String.starts_with?(encoded, "age1")
      assert {:ok, {"age", ^bytes}} = Bech32.decode(encoded)
    end

    test "decodes the BIP-173 vector a12uel5l in either case" do
      assert {:ok, {"a", ""}} = Bech32.decode("a12uel5l")
      assert {:ok, {"a", ""}} = Bech32.decode("A12UEL5L")
    end

    test "rejects mixed case, a missing separator, and a bad checksum" do
      assert :error = Bech32.decode("a12UEL5L")
      assert :error = Bech32.decode("a2uel5l")
      assert :error = Bech32.decode("a12uel5q")
    end

    test "identities are uppercase with the age-secret-key- part" do
      %{identity: identity, private: private} = Support.generate_identity()

      assert String.starts_with?(identity, "AGE-SECRET-KEY-1")
      assert {:ok, {"age-secret-key-", ^private}} = Bech32.decode(identity)
    end
  end
end
