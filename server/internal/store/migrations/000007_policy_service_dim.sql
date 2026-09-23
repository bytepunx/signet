-- BREAKING CHANGE: access policy patterns are now matched against a
-- three-segment target "namespace/service/secret_name" instead of the
-- previous two-segment "namespace/secret_name". Without this, two services
-- in the same namespace with a same-named secret were indistinguishable to a
-- policy, and patterns written in the documented three-segment form (see
-- design/draft.md section 4) never matched anything at all.
--
-- Existing two-segment patterns (exactly one '/') are rewritten to insert a
-- wildcard service segment, e.g. "prod/db-*" -> "prod/*/db-*", preserving
-- "matches any service in this namespace" -- the closest safe equivalent to
-- their prior (non-)behavior. Patterns that already have two or more '/'
-- are left untouched (they were already being interpreted as three-or-more
-- segment patterns by path.Match and are assumed to already target the new
-- shape, or are custom multi-segment globs the operator wrote intentionally).
--
-- Operators should review policies after upgrading: query access_policies
-- directly (there is no 'signet policy' CLI command) and confirm each
-- pattern's middle segment expresses the intended service scope.
UPDATE access_policies
SET secret_pattern = regexp_replace(secret_pattern, '^([^/]*)/([^/].*)$', '\1/*/\2')
WHERE array_length(regexp_split_to_array(secret_pattern, '/'), 1) = 2;
