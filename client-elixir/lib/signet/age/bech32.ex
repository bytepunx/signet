defmodule Signet.Age.Bech32 do
  @moduledoc """
  Bech32 (BIP-173) encoding, as age uses it for `age1...` recipients and
  `AGE-SECRET-KEY-1...` identities.

  Decoding is case-insensitive but rejects mixed case, as the reference
  implementation does. age removes BIP-173's 90-character limit, and so does
  this module.
  """

  import Bitwise

  @charset ~c"qpzry9x8gf2tvdw0s3jn54khce6mua7l"
  @values Map.new(Enum.with_index(@charset))
  @generator [0x3B6A57B2, 0x26508E6D, 0x1EA119FA, 0x3D4233DD, 0x2A1462B3]

  @doc "Encodes `bytes` under the human-readable part `hrp`, in lowercase."
  @spec encode(String.t(), binary()) :: String.t()
  def encode(hrp, bytes) when is_binary(hrp) and is_binary(bytes) do
    {:ok, data} = convert_bits(:binary.bin_to_list(bytes), 8, 5, true)
    values = data ++ checksum(hrp, data)
    hrp <> "1" <> for(v <- values, into: "", do: <<Enum.at(@charset, v)>>)
  end

  @doc "Decodes a Bech32 string into its human-readable part and bytes."
  @spec decode(String.t()) :: {:ok, {String.t(), binary()}} | :error
  def decode(string) when is_binary(string) do
    lower = String.downcase(string)

    if string != lower and string != String.upcase(string) do
      :error
    else
      case List.last(:binary.matches(lower, "1")) do
        {pos, 1} ->
          decode_parts(
            binary_part(lower, 0, pos),
            binary_part(lower, pos + 1, byte_size(lower) - pos - 1)
          )

        nil ->
          :error
      end
    end
  end

  defp decode_parts(hrp, rest) do
    with true <- hrp != "" and byte_size(rest) >= 6,
         true <- Enum.all?(String.to_charlist(hrp), &(&1 in 33..126)),
         {:ok, values} <- to_values(rest),
         true <- polymod(expand(hrp) ++ values) == 1,
         {:ok, bytes} <- convert_bits(Enum.drop(values, -6), 5, 8, false) do
      {:ok, {hrp, :binary.list_to_bin(bytes)}}
    else
      _ -> :error
    end
  end

  defp to_values(chars) do
    chars
    |> String.to_charlist()
    |> Enum.reduce_while({:ok, []}, fn c, {:ok, acc} ->
      case Map.fetch(@values, c) do
        {:ok, v} -> {:cont, {:ok, [v | acc]}}
        :error -> {:halt, :error}
      end
    end)
    |> case do
      {:ok, values} -> {:ok, Enum.reverse(values)}
      :error -> :error
    end
  end

  defp checksum(hrp, data) do
    pm = bxor(polymod(expand(hrp) ++ data ++ [0, 0, 0, 0, 0, 0]), 1)
    for i <- 0..5, do: pm >>> (5 * (5 - i)) &&& 31
  end

  defp expand(hrp) do
    chars = String.to_charlist(hrp)
    Enum.map(chars, &(&1 >>> 5)) ++ [0] ++ Enum.map(chars, &(&1 &&& 31))
  end

  defp polymod(values) do
    Enum.reduce(values, 1, fn v, chk ->
      top = chk >>> 25
      chk = bxor((chk &&& 0x1FFFFFF) <<< 5, v)

      @generator
      |> Enum.with_index()
      |> Enum.reduce(chk, fn {g, i}, chk ->
        if (top >>> i &&& 1) == 1, do: bxor(chk, g), else: chk
      end)
    end)
  end

  # Regroups a list of `from`-bit values into `to`-bit values. With `pad?`
  # the last value is zero-padded (encoding); without it, leftover bits must
  # be fewer than `from` and all zero (decoding), as BIP-173 requires.
  defp convert_bits(values, from, to, pad?) do
    max = (1 <<< to) - 1
    max_acc = (1 <<< (from + to - 1)) - 1

    {acc, bits, out} =
      Enum.reduce(values, {0, 0, []}, fn v, {acc, bits, out} ->
        drain((acc <<< from ||| v) &&& max_acc, bits + from, to, out)
      end)

    cond do
      pad? and bits > 0 -> {:ok, Enum.reverse([acc <<< (to - bits) &&& max | out])}
      pad? -> {:ok, Enum.reverse(out)}
      bits >= from or (acc <<< (to - bits) &&& max) != 0 -> :error
      true -> {:ok, Enum.reverse(out)}
    end
  end

  defp drain(acc, bits, to, out) when bits >= to do
    bits = bits - to
    drain(acc, bits, to, [acc >>> bits &&& (1 <<< to) - 1 | out])
  end

  defp drain(acc, bits, _to, out), do: {acc, bits, out}
end
