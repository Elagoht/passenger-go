package web

import (
	"context"

	"passenger-go/internal/vault"
)

// kdfSlots bounds how many Argon2id derivations (64 MiB each in production)
// the auth forms run at once.
var kdfSlots = make(chan struct{}, 2)

func acquireKDF(ctx context.Context) error {
	select {
	case kdfSlots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func releaseKDF() { <-kdfSlots }

// Wrap is vault.Wrap run in a KDF slot; waiting for one ends with ctx.
func Wrap(ctx context.Context, secret string, dek vault.DEK, params vault.KDFParams, aad string) (vault.Wrapped, error) {
	if err := acquireKDF(ctx); err != nil {
		return vault.Wrapped{}, err
	}
	defer releaseKDF()
	return vault.Wrap(secret, dek, params, aad)
}

// Unwrap is vault.Unwrap run in a KDF slot; waiting for one ends with ctx.
func Unwrap(ctx context.Context, secret string, w vault.Wrapped, params vault.KDFParams, aad string) (vault.DEK, error) {
	if err := acquireKDF(ctx); err != nil {
		return vault.DEK{}, err
	}
	defer releaseKDF()
	return vault.Unwrap(secret, w, params, aad)
}
