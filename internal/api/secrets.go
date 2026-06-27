// Package api contains gRPC service implementations.
package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"strings"

	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/auth"
	"github.com/bytepunx/signet/internal/audit"
	"github.com/bytepunx/signet/internal/store"
	signetv1 "github.com/bytepunx/signet/gen/signet/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

const unknownIdentity = "unauthenticated"

// SecretsServer implements signetv1.SecretsServiceServer.
type SecretsServer struct {
	signetv1.UnimplementedSecretsServiceServer
	store   secretFetcher
	keys    keyUnwrapper
	checker permissionChecker
	audit   auditRecorder
	bus     *Bus
	lockMgr *LockManager
}

// NewSecretsServer constructs a SecretsServer. All parameters are required.
func NewSecretsServer(
	store secretFetcher,
	keys keyUnwrapper,
	checker permissionChecker,
	audit auditRecorder,
	bus *Bus,
	lockMgr *LockManager,
) *SecretsServer {
	return &SecretsServer{store: store, keys: keys, checker: checker, audit: audit, bus: bus, lockMgr: lockMgr}
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
	s.record(ctx, spiffeID, "get", req.Namespace, req.Name, outcome)

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
		s.record(ctx, spiffeID, "watch", req.Namespace, req.Name, "denied")
		return err
	}
	s.record(ctx, spiffeID, "watch", req.Namespace, req.Name, "permitted")

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
				s.record(ctx, spiffeID, "watch", req.Namespace, req.Name, "denied")
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
			s.record(ctx, spiffeID, "watch", req.Namespace, req.Name, "permitted")
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
		return nil, toGRPCError(err)
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
		return nil, toGRPCError(err)
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
		return toGRPCError(err)
	}

	sendCurrent := func() error {
		raw, version, err := s.store.GetServiceConfig(ctx, req.Namespace, req.Service)
		if err != nil {
			if isNotFound(err) {
				return stream.Send(&signetv1.WatchServiceConfigResponse{
					EventType: signetv1.WatchServiceConfigResponse_EVENT_TYPE_DELETED,
					Namespace: req.Namespace,
					Service:   req.Service,
				})
			}
			return toGRPCError(err)
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
		plaintext, decErr := s.decryptSecret(sec.EncryptedDEK, sec.Ciphertext)
		if decErr != nil {
			slog.Error("decrypt secret in bundle", "namespace", req.Namespace, "service", req.Service, "name", sec.Name, "err", decErr)
			return nil, status.Errorf(codes.Internal, "decrypt secret %q: %v", sec.Name, decErr)
		}
		secretsMap[sec.Name] = base64.StdEncoding.EncodeToString(plaintext)
		zeroBytes(plaintext)
	}
	merged["secrets"] = secretsMap

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
		return toGRPCError(err)
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

// decryptSecret unwraps the DEK and decrypts ciphertext, returning plaintext.
// Caller is responsible for zeroing the returned slice when done.
func (s *SecretsServer) decryptSecret(encDEK, ciphertext []byte) ([]byte, error) {
	var plaintext []byte
	if err := s.keys.Use(func(masterKey []byte) error {
		dek, err := icrypto.UnwrapKey(masterKey, encDEK)
		if err != nil {
			return err
		}
		defer zeroBytes(dek)
		plaintext, err = icrypto.Decrypt(dek, ciphertext)
		return err
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

	decryptErr := s.keys.Use(func(masterKey []byte) error {
		dek, unwrapErr := icrypto.UnwrapKey(masterKey, sec.EncryptedDEK)
		if unwrapErr != nil {
			return unwrapErr
		}
		defer zeroBytes(dek)
		var decErr error
		plaintext, decErr = icrypto.Decrypt(dek, sec.Ciphertext)
		return decErr
	})
	if decryptErr != nil {
		err = toGRPCError(decryptErr)
	}
	return
}

func (s *SecretsServer) record(ctx context.Context, spiffeID, action, namespace, name, outcome string) {
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
	}
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
