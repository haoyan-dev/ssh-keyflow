// Package roster holds the fleet roster field rules (.scratch/fleet-sync/roster-format.md).
// State v4 and the node commands use them so a node description saved
// locally is always valid as a roster node.
package roster

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math/big"
	"net/netip"
	"regexp"
	"strings"
)

const (
	RoleClient = "client"
	RoleRemote = "remote"
	MaxKeys    = 16
	// MaxAddresses bounds the addresses of one remote.
	MaxAddresses = 16
	DefaultPort  = 22
)

var (
	label   = `[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?`
	name    = regexp.MustCompile(`^` + label + `$`)
	dnsName = regexp.MustCompile(`^` + label + `(\.` + label + `)*$`)
	account = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]{0,63}$`)
	numeric = regexp.MustCompile(`^[0-9.]+$`)
)

// ValidName reports whether value is a node name: one lowercase DNS label.
func ValidName(value string) bool { return name.MatchString(value) }

// ValidAccount reports whether value is a login account name.
func ValidAccount(value string) bool { return account.MatchString(value) }

// ValidAddress reports whether value is a canonical IPv4 or IPv6 literal or
// a lowercase DNS name, with no port and no brackets.
func ValidAddress(value string) bool {
	if address, err := netip.ParseAddr(value); err == nil {
		return address.Zone() == "" && address.String() == value
	}
	return len(value) <= 253 && dnsName.MatchString(value) && !numeric.MatchString(value)
}

// Roles parses a role list into the fixed order client, remote. Empty,
// unknown and repeated roles are rejected.
func Roles(values []string) ([]string, error) {
	seen := map[string]bool{}
	for _, value := range values {
		if (value != RoleClient && value != RoleRemote) || seen[value] {
			return nil, fmt.Errorf("roles must be client, remote or both, each once")
		}
		seen[value] = true
	}
	var roles []string
	for _, role := range []string{RoleClient, RoleRemote} {
		if seen[role] {
			roles = append(roles, role)
		}
	}
	if len(roles) == 0 {
		return nil, fmt.Errorf("at least one role is required")
	}
	return roles, nil
}

// PublicKey is an OpenSSH public key reduced to "<type> <base64>".
type PublicKey struct {
	Type        string
	Text        string
	Fingerprint string
}

var clientTypes = []string{"ssh-ed25519", "ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521",
	"sk-ssh-ed25519@openssh.com", "sk-ecdsa-sha2-nistp256@openssh.com"}

// ParsePublicKey reads one public key line, dropping any comment. Host keys
// may also be ssh-rsa of at least 2048 bits. DSA keys and certificates are
// rejected.
func ParsePublicKey(line string, host bool) (PublicKey, error) {
	if strings.ContainsAny(strings.TrimSpace(line), "\r\n\x00") {
		return PublicKey{}, fmt.Errorf("expected one public key line")
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return PublicKey{}, fmt.Errorf("expected one public key line")
	}
	keyType := fields[0]
	allowed := false
	for _, candidate := range clientTypes {
		allowed = allowed || candidate == keyType
	}
	if host && keyType == "ssh-rsa" {
		allowed = true
	}
	if !allowed {
		return PublicKey{}, fmt.Errorf("unsupported public key type %q", keyType)
	}
	blob, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil || base64.StdEncoding.EncodeToString(blob) != fields[1] {
		return PublicKey{}, fmt.Errorf("invalid public key encoding")
	}
	parts, err := wireStrings(blob)
	if err != nil || len(parts) == 0 || string(parts[0]) != keyType {
		return PublicKey{}, fmt.Errorf("public key type does not match its data")
	}
	if err := checkShape(keyType, parts); err != nil {
		return PublicKey{}, err
	}
	digest := sha256.Sum256(blob)
	return PublicKey{Type: keyType, Text: keyType + " " + fields[1],
		Fingerprint: "SHA256:" + base64.RawStdEncoding.EncodeToString(digest[:])}, nil
}

func checkShape(keyType string, parts [][]byte) error {
	switch {
	case keyType == "ssh-rsa":
		if len(parts) != 3 {
			return fmt.Errorf("invalid ssh-rsa key")
		}
		if new(big.Int).SetBytes(parts[2]).BitLen() < 2048 {
			return fmt.Errorf("ssh-rsa host keys need at least 2048 bits")
		}
	case keyType == "ssh-ed25519":
		if len(parts) != 2 || len(parts[1]) != 32 {
			return fmt.Errorf("invalid ssh-ed25519 key")
		}
	case keyType == "sk-ssh-ed25519@openssh.com":
		if len(parts) != 3 || len(parts[1]) != 32 {
			return fmt.Errorf("invalid %s key", keyType)
		}
	case strings.HasPrefix(keyType, "ecdsa-sha2-"):
		if len(parts) != 3 || string(parts[1]) != strings.TrimPrefix(keyType, "ecdsa-sha2-") {
			return fmt.Errorf("invalid %s key", keyType)
		}
	case keyType == "sk-ecdsa-sha2-nistp256@openssh.com":
		if len(parts) != 4 || string(parts[1]) != "nistp256" {
			return fmt.Errorf("invalid %s key", keyType)
		}
	}
	return nil
}

// wireStrings splits an SSH wire-format blob into its length-prefixed fields.
func wireStrings(blob []byte) ([][]byte, error) {
	var parts [][]byte
	for len(blob) > 0 {
		if len(blob) < 4 {
			return nil, fmt.Errorf("truncated key data")
		}
		length := binary.BigEndian.Uint32(blob[:4])
		if uint64(length) > uint64(len(blob)-4) {
			return nil, fmt.Errorf("truncated key data")
		}
		parts = append(parts, blob[4:4+length])
		blob = blob[4+length:]
	}
	return parts, nil
}
