package roster

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

const (
	Format = "ssh-keyflow-roster"
	Schema = 1
	// Namespace is the ssh-keygen -Y namespace every roster signature uses.
	Namespace = "ssh-keyflow-roster"
	MaxNodes  = 1000
	MaxBytes  = 1 << 20
	// MaxSignatureBytes bounds an armored roster signature.
	MaxSignatureBytes = 64 << 10
	MaxVersion        = 1<<53 - 1
	// TimeLayout is created_at: UTC with seconds and Z, no fraction.
	TimeLayout = "2006-01-02T15:04:05Z"
	// FutureSkew is how far created_at may be ahead before a warning.
	FutureSkew = 5 * time.Minute
)

// ErrNewerSchema means the roster needs a newer SSH KeyFlow.
var ErrNewerSchema = errors.New("this roster uses a newer schema; upgrade SSH KeyFlow")

// ErrNotCanonical means the bytes parse but are not the canonical form (1.3).
var ErrNotCanonical = errors.New("roster is not in canonical form")

// Node is one roster node (roster-format 1.2), fields in canonical order.
type Node struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Roles        []string `json:"roles"`
	LoginAccount string   `json:"login_account,omitempty"`
	Addresses    []string `json:"addresses,omitempty"`
	Port         int      `json:"port,omitempty"`
	ClientKeys   []string `json:"client_keys,omitempty"`
	HostKeys     []string `json:"host_keys,omitempty"`
}

// Roster is the signed fleet document, fields in canonical order.
type Roster struct {
	Format       string `json:"format"`
	Schema       int    `json:"schema"`
	FleetID      string `json:"fleet_id"`
	Version      int64  `json:"version"`
	CreatedAt    string `json:"created_at"`
	LoginAccount string `json:"login_account"`
	Nodes        []Node `json:"nodes"`
}

// ValidID reports whether value is prefix + "-" + 32 lowercase hex digits.
func ValidID(prefix, value string) bool {
	digest, err := hex.DecodeString(strings.TrimPrefix(value, prefix+"-"))
	return strings.HasPrefix(value, prefix+"-") && err == nil && len(digest) == 16 && strings.ToLower(value) == value
}

// ValidTime reports whether value is a created_at time in TimeLayout.
func ValidTime(value string) bool {
	parsed, err := time.Parse(TimeLayout, value)
	return err == nil && parsed.Format(TimeLayout) == value
}

func sortedUnique(values []string) bool {
	return slices.IsSorted(values) && len(slices.Compact(slices.Clone(values))) == len(values)
}

// Validate checks every rule of roster-format 1.2 that does not depend on bytes.
func (roster Roster) Validate() error {
	if roster.Format != Format {
		return fmt.Errorf("format must be %q", Format)
	}
	if roster.Schema > Schema {
		return ErrNewerSchema
	}
	if roster.Schema != Schema {
		return fmt.Errorf("schema must be %d", Schema)
	}
	if !ValidID("fleet", roster.FleetID) {
		return fmt.Errorf("invalid fleet_id")
	}
	if roster.Version < 1 || roster.Version > MaxVersion {
		return fmt.Errorf("version must be 1 to 2^53-1")
	}
	if !ValidTime(roster.CreatedAt) {
		return fmt.Errorf("created_at must be UTC RFC 3339 with seconds and Z")
	}
	if !ValidAccount(roster.LoginAccount) {
		return fmt.Errorf("invalid common login_account")
	}
	if len(roster.Nodes) == 0 || len(roster.Nodes) > MaxNodes {
		return fmt.Errorf("a roster has 1 to %d nodes", MaxNodes)
	}
	names, keys := map[string]bool{}, map[string]bool{}
	for index, node := range roster.Nodes {
		if index > 0 && roster.Nodes[index-1].ID >= node.ID {
			return fmt.Errorf("nodes must be sorted by id with unique ids")
		}
		if err := node.validate(roster.LoginAccount); err != nil {
			return fmt.Errorf("node %s: %w", node.ID, err)
		}
		if names[node.Name] {
			return fmt.Errorf("node name %q is used twice", node.Name)
		}
		names[node.Name] = true
		for _, key := range slices.Concat(node.ClientKeys, node.HostKeys) {
			if keys[key] {
				return fmt.Errorf("a public key appears on more than one node or list")
			}
			keys[key] = true
		}
	}
	return nil
}

func (node Node) validate(common string) error {
	if !ValidID("node", node.ID) || !ValidName(node.Name) {
		return fmt.Errorf("invalid id or name")
	}
	roles, err := Roles(node.Roles)
	if err != nil || !slices.Equal(roles, node.Roles) {
		return fmt.Errorf("roles must be client, remote or both, in that order")
	}
	client, remote := slices.Contains(roles, RoleClient), slices.Contains(roles, RoleRemote)
	keys := func(list []string, host bool, label string) error {
		if len(list) == 0 || len(list) > MaxKeys || !sortedUnique(list) {
			return fmt.Errorf("%s: 1 to %d unique keys in byte order", label, MaxKeys)
		}
		for _, line := range list {
			if key, err := ParsePublicKey(line, host); err != nil || key.Text != line {
				return fmt.Errorf("%s: invalid key", label)
			}
		}
		return nil
	}
	if client {
		if err := keys(node.ClientKeys, false, "client_keys"); err != nil {
			return err
		}
	} else if node.ClientKeys != nil {
		return fmt.Errorf("client_keys need the client role")
	}
	if !remote {
		if node.LoginAccount != "" || node.Addresses != nil || node.Port != 0 || node.HostKeys != nil {
			return fmt.Errorf("login_account, addresses, port and host_keys need the remote role")
		}
		return nil
	}
	if node.LoginAccount != "" && (!ValidAccount(node.LoginAccount) || node.LoginAccount == common) {
		return fmt.Errorf("login_account must be valid and differ from the common account")
	}
	if len(node.Addresses) == 0 || len(node.Addresses) > MaxAddresses || !sortedUnique(node.Addresses) {
		return fmt.Errorf("addresses: 1 to %d unique entries in byte order", MaxAddresses)
	}
	for _, address := range node.Addresses {
		if !ValidAddress(address) {
			return fmt.Errorf("invalid address %q", address)
		}
	}
	if node.Port == DefaultPort || node.Port < 0 || node.Port > 65535 {
		return fmt.Errorf("port must be 1-65535 and is omitted when 22")
	}
	return keys(node.HostKeys, true, "host_keys")
}

// Marshal returns the canonical bytes (roster-format 1.3) of a valid roster.
func (roster Roster) Marshal() ([]byte, error) {
	if err := roster.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(roster, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if len(data) > MaxBytes {
		return nil, fmt.Errorf("roster exceeds %d bytes", MaxBytes)
	}
	return data, nil
}

// Parse reads canonical roster bytes. It never normalizes: any byte that
// differs from the canonical form is ErrNotCanonical.
func Parse(data []byte) (Roster, error) {
	return parseCanonical(data, "roster", Format, Schema, ErrNotCanonical, Roster.Marshal)
}

// parseCanonical strictly decodes a document of the given format whose only
// accepted form is marshal's output; any other byte is notCanonical.
func parseCanonical[T any](data []byte, label, format string, schema int64, notCanonical error, marshal func(T) ([]byte, error)) (T, error) {
	var zero T
	if len(data) > MaxBytes {
		return zero, fmt.Errorf("%s exceeds %d bytes", label, MaxBytes)
	}
	var header struct {
		Format string          `json:"format"`
		Schema json.RawMessage `json:"schema"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return zero, fmt.Errorf("%s is not a JSON object: %w", label, err)
	}
	if header.Format != format {
		return zero, fmt.Errorf("format must be %q", format)
	}
	var version int64
	if err := json.Unmarshal(header.Schema, &version); err != nil {
		return zero, fmt.Errorf("schema must be an integer")
	}
	if version > schema {
		return zero, ErrNewerSchema
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var document T
	if err := decoder.Decode(&document); err != nil {
		return zero, fmt.Errorf("invalid %s: %w", label, err)
	}
	if decoder.More() {
		return zero, fmt.Errorf("invalid %s: trailing data", label)
	}
	canonical, err := marshal(document)
	if err != nil {
		return zero, fmt.Errorf("invalid %s: %w", label, err)
	}
	if !bytes.Equal(canonical, data) {
		return zero, notCanonical
	}
	return document, nil
}

// NodeFormat and NodeSchema identify a node description file (roster-format 3.1).
const (
	NodeFormat = "ssh-keyflow-node"
	NodeSchema = 1
)

// ErrNodeNotCanonical means node.json parses but is not in canonical form.
var ErrNodeNotCanonical = errors.New("node description file is not in canonical form")

// NodeFile is the unsigned node.json a node exports for the admin node.
type NodeFile struct {
	Format string `json:"format"`
	Schema int    `json:"schema"`
	Node   Node   `json:"node"`
}

// Validate checks the node alone; its login account is compared with the
// common account only when the admin node adds it to a roster.
func (file NodeFile) Validate() error {
	if file.Format != NodeFormat {
		return fmt.Errorf("format must be %q", NodeFormat)
	}
	if file.Schema > NodeSchema {
		return ErrNewerSchema
	}
	if file.Schema != NodeSchema {
		return fmt.Errorf("schema must be %d", NodeSchema)
	}
	if err := file.Node.validate(""); err != nil {
		return fmt.Errorf("node %s: %w", file.Node.ID, err)
	}
	keys := slices.Concat(file.Node.ClientKeys, file.Node.HostKeys)
	slices.Sort(keys)
	if len(slices.Compact(keys)) != len(file.Node.ClientKeys)+len(file.Node.HostKeys) {
		return fmt.Errorf("node %s: a public key appears in more than one list", file.Node.ID)
	}
	return nil
}

// Marshal returns the canonical bytes of a valid node description file.
func (file NodeFile) Marshal() ([]byte, error) {
	if err := file.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// ParseNodeFile reads canonical node.json bytes and never normalizes them.
func ParseNodeFile(data []byte) (NodeFile, error) {
	return parseCanonical(data, "node description file", NodeFormat, NodeSchema, ErrNodeNotCanonical, NodeFile.Marshal)
}

// ValidSignature reports whether text is one armored SSH signature block
// with LF line endings, as ssh-keygen -Y sign writes it.
func ValidSignature(text string) bool {
	const begin, end = "-----BEGIN SSH SIGNATURE-----\n", "-----END SSH SIGNATURE-----\n"
	if !strings.HasPrefix(text, begin) || !strings.HasSuffix(text, end) || len(text) > MaxSignatureBytes {
		return false
	}
	body := strings.TrimSuffix(strings.TrimPrefix(text, begin), end)
	if body == "" || !strings.HasSuffix(body, "\n") {
		return false
	}
	blob, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(body, "\n", ""))
	return err == nil && bytes.HasPrefix(blob, []byte("SSHSIG"))
}

// AgeWarning returns a warning when created_at is older than maxAge or more
// than FutureSkew ahead of now, and "" otherwise.
func AgeWarning(roster Roster, maxAge time.Duration, now time.Time) string {
	created, err := time.Parse(TimeLayout, roster.CreatedAt)
	if err != nil {
		return "roster created_at is unreadable"
	}
	switch age := now.Sub(created); {
	case age > maxAge:
		return fmt.Sprintf("warning: roster version %d was signed %s (%d days ago), older than --max-age %d days; ask the admin node to run fleet sign",
			roster.Version, roster.CreatedAt, int(age.Hours()/24), int(maxAge.Hours()/24))
	case -age > FutureSkew:
		return fmt.Sprintf("warning: roster version %d is dated %s, more than 5 minutes in the future; check the clocks", roster.Version, roster.CreatedAt)
	}
	return ""
}
