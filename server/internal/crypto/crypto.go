// Package crypto owns the envelope encryption hierarchy and all key material.
// No raw key bytes leave this package — callers receive opaque handles via KeyStore.Use.
package crypto
