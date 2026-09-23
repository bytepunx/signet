// Package api contains gRPC service implementations.
package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"

	signetv1 "github.com/bytepunx/signet/gen/signet/v1"
	"github.com/bytepunx/signet/internal/audit"
	"github.com/bytepunx/signet/internal/auth"
	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

const unknownIdentity = "unauthenticated"

// Sentinel audit SecretName values for actions that touch an entire document
// rather than one named secret. audit.Entry requires a non-empty SecretName;
// these make it clear in the audit log what kind of access occurred.
const (
	configAuditName = "<config>"
	bundleAuditName = "<bundle>"
)

// SecretsServer implements signetv1.SecretsServiceServer.
type SecretsServer struct {
	signetv1.UnimplementedSecretsServiceServer
	store   secretFetcher
	keys    keyUnwrapper
	checker permissionChecker
	audit   auditRecorder
	bus     *Bus
	lockMgr *LockManager
	// auditFailClosed, when true, denies access (returns codes.Unavailable)
	// on any permitted operation whose audit log write fails, rather than
	// serving the secret/config/bundle without a durable audit trail.
	auditFailClosed bool
}

// NewSecretsServer constructs a SecretsServer. All parameters are required.
// auditFailClosed controls whether a failed audit write on an otherwise-
// permitted access blocks that access (recommended for production).
func NewSecretsServer(
	store secretFetcher,
	keys keyUnwrapper,
	checker permissionChecker,
	audit auditRecorder,
	bus *Bus,
	lockMgr *LockManager,
	auditFailClosed bool,
) *SecretsServer {
	return &SecretsServer{
		store: store, keys: keys, checker: checker, audit: audit, bus: bus, lockMgr: lockMgr,
		auditFailClosed: auditFailClosed,
	}
}

// GetSecret fetches, decrypts, and returns the latest version of a secret.
// Every call is audited — permitted and denied alike.
func (s *SecretsServer) GetSecret(ctx context.Context, req *signetv1.GetSecretRequest) (*signetv1.GetSecretResponse, error) {
	if req.Namespace == "" || req.Service == "" || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "namespace, service, and name are required")
	}

	spiffeID, plaintext, version, err := s.fetchAndDecrypt(ctx, req.Namespace, req.Service, req.Name)
	outcome := "permitted"
	if err != nil {
		outcome = "denied"
	}
	if auditErr := s.auditOrDeny(ctx, spiffeID, "get", req.Namespace, req.Name, outcome); auditErr != nil {
		return nil, auditErr
	}

	if err != nil {
		return nil, err
	}
	return &signetv1.GetSecretResponse{Value: plaintext, Version: int32(version)}, nil
}

// WatchSecret streams the current value then subsequent updates to the caller.
// The stream runs until the client disconnects or the secret is deleted.
func (s *SecretsServer) WatchSecret(req *signetv1.WatchSecretRequest, stream grpc.ServerStreamingServer[signetv1.WatchSecretResponse]) error {
	ctx := stream.Context()

	if req.Namespace == "" || req.Service == "" || req.Name == "" {
		return status.Error(codes.InvalidArgument, "namespace, service, and name are required")
	}

	// Subscribe before the initial fetch so we cannot miss a write that
	// occurs between the fetch and the select loop.
	ch := s.bus.Subscribe(req.Namespace, req.Service, req.Name)
	defer s.bus.Unsubscribe(req.Namespace, req.Service, req.Name, ch)

	spiffeID, plaintext, version, err := s.fetchAndDecrypt(ctx, req.Namespace, req.Service, req.Name)
	if err != nil {
		_ = s.record(ctx, spiffeID, "watch", req.Namespace, req.Name, "denied")
		return err
	}
	if auditErr := s.auditOrDeny(ctx, spiffeID, "watch", req.Namespace, req.Name, "permitted"); auditErr != nil {
		return auditErr
	}

	if err := stream.Send(&signetv1.WatchSecretResponse{
		EventType: signetv1.WatchSecretResponse_EVENT_TYPE_UPDATED,
		Namespace: req.Namespace,
		Service:   req.Service,
		Name:      req.Name,
		Value:     plaintext,
		Version:   int32(version),
	}); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ch:
			spiffeID, plaintext, version, err = s.fetchAndDecrypt(ctx, req.Namespace, req.Service, req.Name)
			if err != nil {
				_ = s.record(ctx, spiffeID, "watch", req.Namespace, req.Name, "denied")
				if isNotFound(err) {
					// Notify client the secret was removed before closing.
					_ = stream.Send(&signetv1.WatchSecretResponse{
						EventType: signetv1.WatchSecretResponse_EVENT_TYPE_DELETED,
						Namespace: req.Namespace,
						Service:   req.Service,
						Name:      req.Name,
					})
				}
				return err
			}
			if auditErr := s.auditOrDeny(ctx, spiffeID, "watch", req.Namespace, req.Name, "permitted"); auditErr != nil {
				return auditErr
			}
			if err := stream.Send(&signetv1.WatchSecretResponse{
				EventType: signetv1.WatchSecretResponse_EVENT_TYPE_UPDATED,
				Namespace: req.Namespace,
				Service:   req.Service,
				Name:      req.Name,
				Value:     plaintext,
				Version:   int32(version),
			}); err != nil {
				return err
			}
		}
	}
}

// GetConfig returns the value at a dot-path key within a service's config document.
// Access follows the same convention-first policy model as GetSecret.
func (s *SecretsServer) GetConfig(ctx context.Context, req *signetv1.GetConfigRequest) (*signetv1.GetConfigResponse, error) {
	if req.Namespace == "" || req.Service == "" || req.Key == "" {
		return nil, status.Error(codes.InvalidArgument, "namespace, service, and key are required")
	}

	spiffeID, authErr := auth.SPIFFEIDFromContext(ctx)
	if authErr != nil {
		return nil, toGRPCError(authErr)
	}
	if err := s.checker.Allow(ctx, spiffeID, "get", req.Namespace, req.Service, req.Key); err != nil {
		_ = s.record(ctx, spiffeID, "get_config", req.Namespace, req.Key, "denied")
		return nil, toGRPCError(err)
	}
	if auditErr := s.auditOrDeny(ctx, spiffeID, "get_config", req.Namespace, req.Key, "permitted"); auditErr != nil {
		return nil, auditErr
	}

	raw, version, err := s.store.GetServiceConfig(ctx, req.Namespace, req.Service)
	if err != nil {
		return nil, toGRPCError(err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, status.Errorf(codes.Internal, "decode config: %v", err)
	}

	val, ok := navigateDotPath(m, req.Key)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "key %q not found in config for %s/%s", req.Key, req.Namespace, req.Service)
	}

	pbVal, err := structpb.NewValue(val)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode value: %v", err)
	}
	return &signetv1.GetConfigResponse{Value: pbVal, Version: int32(version)}, nil
}

// GetServiceConfig returns the full configuration document for a service.
// Access follows the same convention-first policy model as GetSecret.
func (s *SecretsServer) GetServiceConfig(ctx context.Context, req *signetv1.GetServiceConfigRequest) (*signetv1.GetServiceConfigResponse, error) {
	if req.Namespace == "" || req.Service == "" {
		return nil, status.Error(codes.InvalidArgument, "namespace and service are required")
	}

	spiffeID, authErr := auth.SPIFFEIDFromContext(ctx)
	if authErr != nil {
		return nil, toGRPCError(authErr)
	}
	if err := s.checker.Allow(ctx, spiffeID, "get", req.Namespace, req.Service, ""); err != nil {
		_ = s.record(ctx, spiffeID, "get_service_config", req.Namespace, configAuditName, "denied")
		return nil, toGRPCError(err)
	}
	if auditErr := s.auditOrDeny(ctx, spiffeID, "get_service_config", req.Namespace, configAuditName, "permitted"); auditErr != nil {
		return nil, auditErr
	}

	raw, version, err := s.store.GetServiceConfig(ctx, req.Namespace, req.Service)
	if err != nil {
		return nil, toGRPCError(err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, status.Errorf(codes.Internal, "decode config: %v", err)
	}

	pbStruct, err := structpb.NewStruct(m)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode config: %v", err)
	}
	return &signetv1.GetServiceConfigResponse{Values: pbStruct, Version: int32(version)}, nil
}

// WatchServiceConfig streams the service config document then subsequent updates.
// The stream runs until the client disconnects or the config is deleted.
func (s *SecretsServer) WatchServiceConfig(req *signetv1.WatchServiceConfigRequest, stream grpc.ServerStreamingServer[signetv1.WatchServiceConfigResponse]) error {
	ctx := stream.Context()

	if req.Namespace == "" || req.Service == "" {
		return status.Error(codes.InvalidArgument, "namespace and service are required")
	}

	ch := s.bus.SubscribeService(req.Namespace, req.Service)
	defer s.bus.UnsubscribeService(req.Namespace, req.Service, ch)

	spiffeID, authErr := auth.SPIFFEIDFromContext(ctx)
	if authErr != nil {
		return toGRPCError(authErr)
	}
	if err := s.checker.Allow(ctx, spiffeID, "get", req.Namespace, req.Service, ""); err != nil {
		_ = s.record(ctx, spiffeID, "watch_service_config", req.Namespace, configAuditName, "denied")
		return toGRPCError(err)
	}

	sendCurrent := func() error {
		raw, version, err := s.store.GetServiceConfig(ctx, req.Namespace, req.Service)
		if err != nil {
			if isNotFound(err) {
				_ = s.record(ctx, spiffeID, "watch_service_config", req.Namespace, configAuditName, "denied")
				return stream.Send(&signetv1.WatchServiceConfigResponse{
					EventType: signetv1.WatchServiceConfigResponse_EVENT_TYPE_DELETED,
					Namespace: req.Namespace,
					Service:   req.Service,
				})
			}
			return toGRPCError(err)
		}
		if auditErr := s.auditOrDeny(ctx, spiffeID, "watch_service_config", req.Namespace, configAuditName, "permitted"); auditErr != nil {
			return auditErr
		}
		var m map[string]interface{}
		if err := json.Unmarshal(raw, &m); err != nil {
			return status.Errorf(codes.Internal, "decode config: %v", err)
		}
		pbStruct, err := structpb.NewStruct(m)
		if err != nil {
			return status.Errorf(codes.Internal, "encode config: %v", err)
		}
		return stream.Send(&signetv1.WatchServiceConfigResponse{
			EventType: signetv1.WatchServiceConfigResponse_EVENT_TYPE_UPDATED,
			Namespace: req.Namespace,
			Service:   req.Service,
			Values:    pbStruct,
			Version:   int32(version),
		})
	}

	if err := sendCurrent(); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ch:
			if err := sendCurrent(); err != nil {
				return err
			}
		}
	}
}

// GetServiceBundle returns config and all secrets merged into a single structured
// document. Config keys sit at the top level; secrets are base64-encoded under
// the reserved "secrets" key. If no config exists, only "secrets" is present.
// If no secrets exist, "secrets" is an empty object.
func (s *SecretsServer) GetServiceBundle(ctx context.Context, req *signetv1.GetServiceBundleRequest) (*signetv1.GetServiceBundleResponse, error) {
	if req.Namespace == "" || req.Service == "" {
		return nil, status.Error(codes.InvalidArgument, "namespace and service are required")
	}

	spiffeID, authErr := auth.SPIFFEIDFromContext(ctx)
	if authErr != nil {
		return nil, toGRPCError(authErr)
	}
	if err := s.checker.Allow(ctx, spiffeID, "get", req.Namespace, req.Service, ""); err != nil {
		_ = s.record(ctx, spiffeID, "get_bundle", req.Namespace, bundleAuditName, "denied")
		return nil, toGRPCError(err)
	}

	// Build the merged map starting from config (may be absent).
	merged := make(map[string]interface{})
	configVersion := 0

	raw, version, configErr := s.store.GetServiceConfig(ctx, req.Namespace, req.Service)
	if configErr == nil {
		var m map[string]interface{}
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, status.Errorf(codes.Internal, "decode config: %v", err)
		}
		if _, reserved := m["secrets"]; reserved {
			slog.Warn("config key 'secrets' is reserved and will be overwritten by bundle",
				"namespace", req.Namespace, "service", req.Service)
		}
		for k, v := range m {
			merged[k] = v
		}
		configVersion = version
	} else if !errors.Is(configErr, store.ErrNotFound) {
		return nil, toGRPCError(configErr)
	}

	// Decrypt all secrets and populate the "secrets" sub-map.
	secretsMap := make(map[string]interface{})

	secs, err := s.store.FetchServiceSecrets(ctx, req.Namespace, req.Service)
	if err != nil {
		return nil, toGRPCError(err)
	}

	for _, sec := range secs {
		plaintext, decErr := s.decryptSecret(ctx, sec)
		if decErr != nil {
			slog.Error("decrypt secret in bundle", "namespace", req.Namespace, "service", req.Service, "name", sec.Name, "err", decErr)
			return nil, status.Errorf(codes.Internal, "decrypt secret %q: %v", sec.Name, decErr)
		}
		secretsMap[sec.Name] = base64.StdEncoding.EncodeToString(plaintext)
		zeroBytes(plaintext)
	}
	merged["secrets"] = secretsMap

	// This call decrypts and returns every secret for the service in one
	// shot — the highest-exposure read path — so its audit entry must be
	// durable before the bundle is returned.
	if auditErr := s.auditOrDeny(ctx, spiffeID, "get_bundle", req.Namespace, bundleAuditName, "permitted"); auditErr != nil {
		return nil, auditErr
	}

	pbStruct, err := structpb.NewStruct(merged)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode bundle: %v", err)
	}
	return &signetv1.GetServiceBundleResponse{
		Bundle:        pbStruct,
		ConfigVersion: int32(configVersion),
	}, nil
}

// WatchServiceBundle streams change notifications when any secret or config for
// the service changes. No values are sent in notifications — the intended client
// pattern is to initiate a coordinated shutdown and call GetServiceBundle fresh
// on the next startup. No event is sent on initial connection.
func (s *SecretsServer) WatchServiceBundle(req *signetv1.WatchServiceBundleRequest, stream grpc.ServerStreamingServer[signetv1.WatchServiceBundleResponse]) error {
	ctx := stream.Context()

	if req.Namespace == "" || req.Service == "" {
		return status.Error(codes.InvalidArgument, "namespace and service are required")
	}

	spiffeID, authErr := auth.SPIFFEIDFromContext(ctx)
	if authErr != nil {
		return toGRPCError(authErr)
	}
	if err := s.checker.Allow(ctx, spiffeID, "get", req.Namespace, req.Service, ""); err != nil {
		_ = s.record(ctx, spiffeID, "watch_bundle", req.Namespace, bundleAuditName, "denied")
		return toGRPCError(err)
	}
	// No secret or config values flow through this stream — only a
	// change-notification flag — so one audit entry at subscription time is
	// sufficient; there is no per-notification data access to record.
	if auditErr := s.auditOrDeny(ctx, spiffeID, "watch_bundle", req.Namespace, bundleAuditName, "permitted"); auditErr != nil {
		return auditErr
	}

	ch := s.bus.SubscribeBundle(req.Namespace, req.Service)
	defer s.bus.UnsubscribeBundle(req.Namespace, req.Service, ch)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ch:
			if err := stream.Send(&signetv1.WatchServiceBundleResponse{
				EventType: signetv1.WatchServiceBundleResponse_EVENT_TYPE_CHANGED,
				Namespace: req.Namespace,
				Service:   req.Service,
			}); err != nil {
				return err
			}
		}
	}
}

// decryptSecret unwraps sec's DEK and decrypts its ciphertext, returning
// plaintext. Both the DEK wrap and the ciphertext are bound via AAD to the
// secret's (namespace, service, name) identity so a blob copied from another
// row by a party with database write access (but no key material) fails
// authentication instead of silently decrypting.
//
// When sec.KEKID is set, the DEK is wrapped under that key-encryption-key
// (looked up and unwrapped under the master key); when empty, the secret
// predates the KEK tier and its DEK is wrapped directly under the master key.
// Both branches fall back to the legacy nil-AAD form for data written before
// AAD binding existed, logging a warning so operators can identify secrets
// that still need a re-sync to gain the new binding.
// Caller is responsible for zeroing the returned slice when done.
func (s *SecretsServer) decryptSecret(ctx context.Context, sec store.Secret) ([]byte, error) {
	aad := icrypto.BindAAD(icrypto.AADSecret, sec.Namespace, sec.Service, sec.Name)

	if sec.KEKID == "" {
		var plaintext []byte
		if err := s.keys.Use(func(masterKey []byte) error {
			dek, dekLegacy, err := icrypto.UnwrapKeyWithFallback(masterKey, sec.EncryptedDEK, aad)
			if err != nil {
				return err
			}
			defer zeroBytes(dek)
			pt, ctLegacy, err := icrypto.DecryptWithFallback(dek, sec.Ciphertext, aad)
			if err != nil {
				return err
			}
			if dekLegacy || ctLegacy {
				slog.Warn("secret decrypted via legacy pre-AAD/pre-KEK fallback; re-sync to upgrade encryption",
					"namespace", sec.Namespace, "service", sec.Service, "name", sec.Name)
			}
			plaintext = pt
			return nil
		}); err != nil {
			return nil, err
		}
		return plaintext, nil
	}

	kekRec, err := s.store.GetKEKByID(ctx, sec.KEKID)
	if err != nil {
		return nil, fmt.Errorf("load kek %s: %w", sec.KEKID, err)
	}

	var plaintext []byte
	if err := s.keys.Use(func(masterKey []byte) error {
		kek, err := icrypto.UnwrapKey(masterKey, kekRec.WrappedKEK, icrypto.BindAAD(icrypto.AADKEK))
		if err != nil {
			return fmt.Errorf("unwrap kek: %w", err)
		}
		defer zeroBytes(kek)
		dek, err := icrypto.UnwrapKey(kek, sec.EncryptedDEK, aad)
		if err != nil {
			return err
		}
		defer zeroBytes(dek)
		pt, decErr := icrypto.Decrypt(dek, sec.Ciphertext, aad)
		plaintext = pt
		return decErr
	}); err != nil {
		return nil, err
	}
	return plaintext, nil
}

// navigateDotPath follows a dot-separated key path through a nested map.
// "db.host" on {"db": {"host": "pg"}} returns ("pg", true).
func navigateDotPath(m map[string]interface{}, key string) (interface{}, bool) {
	idx := strings.IndexByte(key, '.')
	if idx < 0 {
		val, ok := m[key]
		return val, ok
	}
	head, tail := key[:idx], key[idx+1:]
	nested, ok := m[head]
	if !ok {
		return nil, false
	}
	nm, ok := nested.(map[string]interface{})
	if !ok {
		return nil, false
	}
	return navigateDotPath(nm, tail)
}

// fetchAndDecrypt performs the full auth → policy → fetch → decrypt pipeline.
// It always returns the authenticated SPIFFE ID so the caller can audit even on failure.
func (s *SecretsServer) fetchAndDecrypt(ctx context.Context, namespace, service, name string) (spiffeID string, plaintext []byte, version int, err error) {
	spiffeID, authErr := auth.SPIFFEIDFromContext(ctx)
	if authErr != nil {
		err = toGRPCError(authErr)
		return
	}

	if policyErr := s.checker.Allow(ctx, spiffeID, "get", namespace, service, name); policyErr != nil {
		err = toGRPCError(policyErr)
		return
	}

	sec, fetchErr := s.store.GetSecret(ctx, namespace, service, name)
	if fetchErr != nil {
		err = toGRPCError(fetchErr)
		return
	}
	version = sec.Version

	plaintext, decErr := s.decryptSecret(ctx, *sec)
	if decErr != nil {
		err = toGRPCError(decErr)
	}
	return
}

// record writes an audit entry and returns the write error, if any (also
// logged). Callers that need fail-closed behavior should use auditOrDeny
// instead of calling record directly.
func (s *SecretsServer) record(ctx context.Context, spiffeID, action, namespace, name, outcome string) error {
	if spiffeID == "" {
		spiffeID = unknownIdentity
	}
	if err := s.audit.Record(ctx, audit.Entry{
		SPIFFEID:   spiffeID,
		Action:     action,
		Namespace:  namespace,
		SecretName: name,
		Outcome:    outcome,
		PeerIP:     peerIP(ctx),
	}); err != nil {
		slog.Error("audit write failed", "action", action, "namespace", namespace, "name", name, "err", err)
		return err
	}
	return nil
}

// auditOrDeny records the access and, when the access was otherwise permitted
// but the audit write itself failed, enforces fail-closed behavior (if
// enabled): the caller must not serve the secret/config/bundle without a
// durable audit trail. Returns a non-nil error only in that fail-closed case;
// callers should return it immediately without also returning the original
// operation error.
func (s *SecretsServer) auditOrDeny(ctx context.Context, spiffeID, action, namespace, name, outcome string) error {
	auditErr := s.record(ctx, spiffeID, action, namespace, name, outcome)
	if outcome == "permitted" && auditErr != nil && s.auditFailClosed {
		return status.Error(codes.Unavailable, "audit log write failed; access denied (fail-closed)")
	}
	return nil
}

func peerIP(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok || p.Addr == nil {
		return ""
	}
	addr := p.Addr.String()
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func isNotFound(err error) bool {
	return status.Code(err) == codes.NotFound
}
