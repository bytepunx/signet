package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/bytepunx/signet/internal/auth"
	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/store"
	"github.com/bytepunx/signet/internal/unseal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AdminServer implements adminv1.AdminServiceServer.
// Every method requires a valid Kubernetes SA bearer token in the gRPC metadata.
type AdminServer struct {
	adminv1.UnimplementedAdminServiceServer
	manager   unsealMgr
	validator tokenChecker
	store     adminStore
	keys      keyUnwrapper
}

// NewAdminServer constructs an AdminServer. store and keys back the
// key-check value verified on every unseal, and the KEK/master-key rotation
// operations.
func NewAdminServer(manager unsealMgr, validator tokenChecker, store adminStore, keys keyUnwrapper) *AdminServer {
	return &AdminServer{manager: manager, validator: validator, store: store, keys: keys}
}

// requireToken extracts and validates the bearer SA token from gRPC metadata.
func (s *AdminServer) requireToken(ctx context.Context) error {
	token, err := auth.TokenFromMetadata(ctx)
	if err != nil {
		return toGRPCError(err)
	}
	if err := s.validator.Validate(ctx, token); err != nil {
		return toGRPCError(err)
	}
	return nil
}

// finishUnseal verifies the just-loaded master key against the stored
// key-check value (minting one on a brand new deployment), sealing the
// server again on mismatch so a wrong key never appears to have succeeded.
func (s *AdminServer) finishUnseal(ctx context.Context) error {
	if err := VerifyOrInitKeyCheckValue(ctx, s.store, s.keys); err != nil {
		s.manager.Seal()
		if errors.Is(err, ErrKeyCheckFailed) {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		return status.Errorf(codes.Internal, "key check: %v", err)
	}
	return nil
}

// UnsealKey unseals the server with a direct 32-byte master key.
func (s *AdminServer) UnsealKey(ctx context.Context, req *adminv1.UnsealKeyRequest) (*adminv1.UnsealKeyResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}
	if len(req.Key) == 0 {
		return nil, status.Error(codes.InvalidArgument, "key must not be empty")
	}
	if err := s.manager.UnsealWithKey(req.Key); err != nil {
		return nil, toGRPCError(err)
	}
	if err := s.finishUnseal(ctx); err != nil {
		return nil, err
	}
	st := s.manager.Status()
	return &adminv1.UnsealKeyResponse{
		Unsealed:       st.State == unseal.StateUnsealed,
		SharesReceived: int32(st.SharesReceived),
		SharesRequired: int32(st.SharesRequired),
		Message:        "unsealed successfully",
	}, nil
}

// UnsealShare submits one Shamir share. The server unseals when the threshold is met.
func (s *AdminServer) UnsealShare(ctx context.Context, req *adminv1.UnsealShareRequest) (*adminv1.UnsealShareResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}
	if len(req.Share) == 0 {
		return nil, status.Error(codes.InvalidArgument, "share must not be empty")
	}
	st, err := s.manager.SubmitShare(req.Share)
	if err != nil {
		return nil, toGRPCError(err)
	}

	if st.State == unseal.StateUnsealed {
		// finishUnseal never changes state on success — a failed key check
		// calls Seal() and returns an error immediately, so there is nothing
		// to re-fetch here.
		if err := s.finishUnseal(ctx); err != nil {
			return nil, err
		}
	}

	msg := fmt.Sprintf("share accepted (%d/%d)", st.SharesReceived, st.SharesRequired)
	if st.State == unseal.StateUnsealed {
		msg = "unsealed successfully"
	}
	return &adminv1.UnsealShareResponse{
		Unsealed:       st.State == unseal.StateUnsealed,
		SharesReceived: int32(st.SharesReceived),
		SharesRequired: int32(st.SharesRequired),
		Message:        msg,
	}, nil
}

// Seal immediately wipes the master key from memory and transitions to sealed.
func (s *AdminServer) Seal(ctx context.Context, _ *adminv1.SealRequest) (*adminv1.SealResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}
	s.manager.Seal()
	return &adminv1.SealResponse{Message: "sealed"}, nil
}

// Status returns the current seal state and Shamir share progress.
func (s *AdminServer) Status(ctx context.Context, _ *adminv1.StatusRequest) (*adminv1.StatusResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}
	st := s.manager.Status()
	return &adminv1.StatusResponse{
		State:          toProtoState(st.State),
		SharesReceived: int32(st.SharesReceived),
		SharesRequired: int32(st.SharesRequired),
	}, nil
}

// RotateKEK generates a new key-encryption-key, deactivates the current one
// (retained for decryption until pruned), and re-wraps every secret's DEK
// from the old KEK to the new one. Secrets predating the KEK tier (direct
// master-key wrap) and secrets already on some other KEK are untouched. On a
// fresh deployment with no active KEK yet, this simply provisions the first one.
func (s *AdminServer) RotateKEK(ctx context.Context, _ *adminv1.RotateKEKRequest) (*adminv1.RotateKEKResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}

	oldRec, oldErr := s.store.GetActiveKEK(ctx)
	if oldErr != nil && !errors.Is(oldErr, store.ErrNotFound) {
		return nil, toGRPCError(oldErr)
	}

	var oldKEKBytes []byte
	if oldErr == nil {
		if err := s.keys.Use(func(masterKey []byte) error {
			plain, uErr := icrypto.UnwrapKey(masterKey, oldRec.WrappedKEK, icrypto.BindAAD(icrypto.AADKEK))
			oldKEKBytes = plain
			return uErr
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "unwrap current kek: %v", err)
		}
		defer zeroBytes(oldKEKBytes)
	}

	newKEKBytes, err := icrypto.GenerateKey()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate kek: %v", err)
	}
	defer zeroBytes(newKEKBytes)

	var newWrapped []byte
	if err := s.keys.Use(func(masterKey []byte) error {
		w, wErr := icrypto.WrapKey(masterKey, newKEKBytes, icrypto.BindAAD(icrypto.AADKEK))
		newWrapped = w
		return wErr
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "wrap new kek: %v", err)
	}

	newRec := &store.KEK{WrappedKEK: newWrapped, IsActive: true}
	if err := s.store.PutKEK(ctx, newRec); err != nil {
		return nil, toGRPCError(err)
	}

	var oldID string
	if oldErr == nil {
		oldID = oldRec.ID
		if err := s.store.DeactivateKEK(ctx, oldRec.ID); err != nil {
			return nil, toGRPCError(err)
		}
	}

	rewrapped := 0
	if oldErr == nil {
		refs, err := s.store.ListSecretKeyRefs(ctx)
		if err != nil {
			return nil, toGRPCError(err)
		}
		for _, ref := range refs {
			if ref.KEKID != oldRec.ID {
				continue
			}
			aad := icrypto.BindAAD(icrypto.AADSecret, ref.Namespace, ref.Service, ref.Name)
			dek, uErr := icrypto.UnwrapKey(oldKEKBytes, ref.EncryptedDEK, aad)
			if uErr != nil {
				slog.Error("rotate kek: unwrap dek failed, leaving on old kek",
					"namespace", ref.Namespace, "service", ref.Service, "name", ref.Name, "err", uErr)
				continue
			}
			newEncDEK, wErr := icrypto.WrapKey(newKEKBytes, dek, aad)
			zeroBytes(dek)
			if wErr != nil {
				slog.Error("rotate kek: rewrap dek failed, leaving on old kek",
					"namespace", ref.Namespace, "service", ref.Service, "name", ref.Name, "err", wErr)
				continue
			}
			if uErr := s.store.UpdateSecretDEK(ctx, ref.Namespace, ref.Service, ref.Name, ref.Version, newEncDEK, newRec.ID); uErr != nil {
				slog.Error("rotate kek: persist rewrap failed, leaving on old kek",
					"namespace", ref.Namespace, "service", ref.Service, "name", ref.Name, "err", uErr)
				continue
			}
			rewrapped++
		}
	}

	return &adminv1.RotateKEKResponse{
		NewKekId:         newRec.ID,
		OldKekId:         oldID,
		SecretsRewrapped: int32(rewrapped),
	}, nil
}

// ListKEKs returns every KEK, active or retained-for-decryption, without
// exposing any key material.
func (s *AdminServer) ListKEKs(ctx context.Context, _ *adminv1.ListKEKsRequest) (*adminv1.ListKEKsResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}
	keks, err := s.store.ListKEKs(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	infos := make([]*adminv1.KEKInfo, len(keks))
	for i, k := range keks {
		info := &adminv1.KEKInfo{
			Id:        k.ID,
			IsActive:  k.IsActive,
			CreatedAt: k.CreatedAt.UTC().Format(time.RFC3339),
		}
		if k.DeactivatedAt != nil {
			info.DeactivatedAt = k.DeactivatedAt.UTC().Format(time.RFC3339)
		}
		infos[i] = info
	}
	return &adminv1.ListKEKsResponse{Keks: infos}, nil
}

// PruneKEK permanently deletes an inactive KEK. Refuses to delete the active
// KEK or one still referenced by any secret's DEK wrap.
func (s *AdminServer) PruneKEK(ctx context.Context, req *adminv1.PruneKEKRequest) (*adminv1.PruneKEKResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}
	id := req.GetId()
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "id must not be empty")
	}

	if active, err := s.store.GetActiveKEK(ctx); err == nil && active.ID == id {
		return nil, status.Error(codes.FailedPrecondition, "cannot prune the active kek; rotate first")
	}

	n, err := s.store.CountSecretsUsingKEK(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}
	if n > 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "cannot prune kek %s: still referenced by %d secret(s)", id, n)
	}

	if err := s.store.DeleteKEK(ctx, id); err != nil {
		return nil, toGRPCError(err)
	}
	return &adminv1.PruneKEKResponse{Message: fmt.Sprintf("kek %s pruned", id)}, nil
}

// RotateMasterKey re-wraps every KEK and the key-check value under a new
// master key, then adopts that key as the server's active master key. It
// never touches secrets or their DEKs directly — only their KEK layer, which
// is why the KEK tier makes master key rotation cheap. The DB update and the
// in-memory key swap are ordered so a failure never leaves the server holding
// a master key that cannot unwrap its own KEKs.
func (s *AdminServer) RotateMasterKey(ctx context.Context, req *adminv1.RotateMasterKeyRequest) (*adminv1.RotateMasterKeyResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}
	if len(req.GetNewKey()) != icrypto.KeySize {
		return nil, status.Errorf(codes.InvalidArgument, "new_key must be exactly %d bytes", icrypto.KeySize)
	}
	newKey := make([]byte, len(req.GetNewKey()))
	copy(newKey, req.GetNewKey())
	defer zeroBytes(newKey)

	keks, err := s.store.ListKEKs(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}

	// The key-check value must already exist: reaching this RPC requires the
	// server to be Unsealed, and every successful unseal mints one if absent.
	// Its absence here means something is already inconsistent; refuse to
	// rotate rather than compound the problem.
	oldKCV, err := s.store.GetKeyCheckValue(ctx)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "cannot rotate master key: %v", err)
	}

	// Keep the pre-rotation wraps so a failed in-memory adoption (below) can
	// be rolled back in the database.
	oldUpdates := make([]store.KEKRewrap, 0, len(keks))
	for _, k := range keks {
		oldUpdates = append(oldUpdates, store.KEKRewrap{ID: k.ID, WrappedKEK: k.WrappedKEK})
	}

	updates := make([]store.KEKRewrap, 0, len(keks))
	var newKCV []byte
	if useErr := s.keys.Use(func(oldMasterKey []byte) error {
		for _, k := range keks {
			plain, uErr := icrypto.UnwrapKey(oldMasterKey, k.WrappedKEK, icrypto.BindAAD(icrypto.AADKEK))
			if uErr != nil {
				return fmt.Errorf("unwrap kek %s: %w", k.ID, uErr)
			}
			newWrapped, wErr := icrypto.WrapKey(newKey, plain, icrypto.BindAAD(icrypto.AADKEK))
			zeroBytes(plain)
			if wErr != nil {
				return fmt.Errorf("rewrap kek %s: %w", k.ID, wErr)
			}
			updates = append(updates, store.KEKRewrap{ID: k.ID, WrappedKEK: newWrapped})
		}

		ct, encErr := icrypto.Encrypt(newKey, []byte(kcvPlaintext), icrypto.BindAAD(icrypto.AADKeyCheckValue))
		if encErr != nil {
			return fmt.Errorf("create new key-check value: %w", encErr)
		}
		newKCV = ct
		return nil
	}); useErr != nil {
		return nil, status.Errorf(codes.Internal, "rewrap key material: %v", useErr)
	}

	if err := s.store.RewrapKEKsAndKCV(ctx, updates, newKCV); err != nil {
		return nil, toGRPCError(err)
	}

	// Only adopt the new key in memory after the DB commit succeeds — the
	// manager zeroes its argument, so hand it a fresh copy.
	rotateKey := make([]byte, len(newKey))
	copy(rotateKey, newKey)
	if err := s.manager.RotateMasterKey(rotateKey); err != nil {
		// The DB now has every KEK (and the KCV) wrapped under newKey, but the
		// in-memory master key is still the old one — nothing can be
		// decrypted until this is resolved. Best-effort roll the DB back to
		// the pre-rotation wraps so the still-loaded old key stays authoritative.
		if rbErr := s.store.RewrapKEKsAndKCV(ctx, oldUpdates, oldKCV); rbErr != nil {
			slog.Error("rotate master key: adopting the new key failed AND rollback failed; "+
				"KEKs are now wrapped under a key not loaded in memory — manual recovery required",
				"adopt_err", err, "rollback_err", rbErr)
		} else {
			slog.Warn("rotate master key: adopting the new key failed; rolled back KEK/KCV wraps to the previous master key", "err", err)
		}
		return nil, toGRPCError(err)
	}

	return &adminv1.RotateMasterKeyResponse{
		Message:       "master key rotated; redistribute the new key to keyholders (Shamir) or update the cluster Secret (auto-unseal) as appropriate",
		KeksRewrapped: int32(len(updates)),
	}, nil
}

func toProtoState(s unseal.State) adminv1.StatusResponse_State {
	switch s {
	case unseal.StateSealed:
		return adminv1.StatusResponse_STATE_SEALED
	case unseal.StateUnsealing:
		return adminv1.StatusResponse_STATE_UNSEALING
	case unseal.StateUnsealed:
		return adminv1.StatusResponse_STATE_UNSEALED
	default:
		return adminv1.StatusResponse_STATE_UNSPECIFIED
	}
}
