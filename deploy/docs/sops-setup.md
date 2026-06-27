# SOPS Setup for signet

This guide walks through encrypting secrets with SOPS and age so they can be
synced into signet from a git repository.

## Prerequisites

- `sops` ≥ 3.8 installed (`brew install sops` / `apt install sops` / binary from [getsops/sops](https://github.com/getsops/sops/releases))
- signet running and unsealed
- `signet` CLI configured (`signet config set server http://localhost:8444`)

## 1. Generate an age keypair in signet

signet manages its own age keypair. On first use, rotate to create one:

```bash
signet sops-key rotate
```

Output:
```
New public key:  age1abc123...
New fingerprint: a1b2c3d4e5f6a1b2
```

The private key is stored encrypted in signet's database and is never exposed.
The public key is the only value you need to configure SOPS.

## 2. Create a .sops.yaml in your secrets repository

In the root of your secrets repository, create a `.sops.yaml` file that tells
SOPS which key to encrypt to:

```yaml
# .sops.yaml
creation_rules:
  - path_regex: secrets/.*\.yaml$
    age: age1abc123...   # paste the public key from step 1
```

All files matching `secrets/**/*.yaml` will be encrypted to signet's key.
If you rotate the key later, re-run `sops rotate` on each file (see Section 5).

## 3. Secret file format

Each secret is a single YAML file with one top-level `value` key:

```yaml
value: the-actual-secret-here
```

The file path determines how signet stores it:

```
secrets/<namespace>/<service>/<name>.yaml
```

Examples:
```
secrets/payments/api/stripe-key.yaml      → namespace=payments, service=api, name=stripe-key
secrets/infra/redis/password.yaml         → namespace=infra,    service=redis, name=password
secrets/platform/auth/jwt-signing-key.yaml→ namespace=platform, service=auth,  name=jwt-signing-key
```

## 4. Encrypt a secret

```bash
# Create the plaintext file
cat > secrets/payments/api/stripe-key.yaml <<'EOF'
value: sk_live_...
EOF

# Encrypt in place
sops --encrypt --in-place secrets/payments/api/stripe-key.yaml

# Verify the file is now encrypted (value field should be ENC[...])
cat secrets/payments/api/stripe-key.yaml
```

The encrypted file is safe to commit to git. The plaintext is only available
to processes that can reach signet and have a policy granting access.

## 5. Key rotation

When you run `signet sops-key rotate`, signet deactivates the old key (kept for
decryption) and generates a new one. You must re-encrypt all SOPS files to the
new key before the old one is pruned.

```bash
# Get the new public key
NEW_KEY=$(signet sops-key get | awk '/Public key:/ {print $3}')

# Update .sops.yaml with the new key
sed -i "s/age: age1.*/age: $NEW_KEY/" .sops.yaml

# Re-encrypt all secrets (requires both keys in .sops.yaml during transition)
find secrets/ -name '*.yaml' | xargs -I{} sops --rotate --in-place {}

# Commit and push — signet will re-sync using the new key
git add -A && git commit -m "rotate sops key" && git push

# Once synced, prune the old key
OLD_KEY=$(signet sops-key list | awk '/inactive/ {print $2}')
signet sops-key prune --public-key "$OLD_KEY"
```

signet retains all inactive keys until explicitly pruned, so files encrypted to
old keys continue to be synced correctly during the rotation window.

## 6. Verify

After pushing, check that signet picked up the change:

```bash
# Trigger an immediate sync (optional — webhook does this automatically)
REPO_ID=$(signet repo list | awk 'NR==2 {print $1}')
signet repo sync --id "$REPO_ID"

# The secret is now available to workloads with a matching policy
```
