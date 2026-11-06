package app_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
)

func TestLocalDeletionResumesCrashAfterFileRemovalWithoutChangingIDs(testingContext *testing.T) {
	for _, checkpoint := range []string{"before public checkpoint", "after public checkpoint", "both files absent"} {
		testingContext.Run(checkpoint, func(testingContext *testing.T) {
			home, external := prepareRegistration(testingContext, true)
			before, _ := state.Load(home)
			if err := state.Update(home, func(document *state.State) error {
				document.Registrations[0].Status, document.Registrations[0].Stage, document.Registrations[0].Operation = "removed", "complete", "unregister"
				document.Deletions = []state.LocalDeletion{{ID: "deletion-fixture", KeyID: document.Keys[0].ID, Stage: "pending", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), PublicSHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(publicKey))), PrivateSHA256: document.Keys[0].PrivateSHA256, PublicRemoved: checkpoint == "after public checkpoint"}}
				return nil
			}); err != nil {
				testingContext.Fatal(err)
			}
			if err := os.Remove(before.Keys[0].PublicPath); err != nil {
				testingContext.Fatal(err)
			}
			if checkpoint == "both files absent" {
				if err := os.Remove(before.Keys[0].PrivatePath); err != nil {
					testingContext.Fatal(err)
				}
			}
			output, err := runCLI(testingContext, home, "6\n1\nyes\n0\n", external)
			after, loadErr := state.Load(home)
			if err != nil || loadErr != nil || len(after.Keys) != 0 || len(after.DeletedKeys) != 1 || after.DeletedKeys[0].Key.ID != before.Keys[0].ID || len(after.Deletions) != 1 || after.Deletions[0].ID != "deletion-fixture" || after.Deletions[0].Stage != "complete" {
				testingContext.Fatal("resume lost identity or journal", err, loadErr, after, output)
			}
		})
	}
}

func TestCreatedKeyDeletionRetainsCompletedCreationJournal(testingContext *testing.T) {
	home := fixtureTempDir(testingContext)
	external := onboardingRunner{testingContext: testingContext, home: home}
	output, err := runCLI(testingContext, home, "5\nc\nindependent\nyes\n6\n1\nyes\n2\n0\n", external)
	document, loadErr := state.Load(home)
	if err != nil || loadErr != nil || len(document.Keys) != 0 || len(document.DeletedKeys) != 1 || len(document.Creations) != 1 || document.Creations[0].KeyID != document.DeletedKeys[0].Key.ID || document.Creations[0].Stage != "complete" {
		testingContext.Fatal("created-key journal invalidated by deletion", err, loadErr, document, output)
	}
}

func TestRevocationAndLocalDeletionPreserveIncludeAndBackupJournal(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, true)
	configPath := writeFixture(testingContext, home, ".ssh/config", "Host *\r\n  ServerAliveInterval 30\r\n")
	output, err := runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\nyes\nyes\n0\n", external)
	if err != nil {
		testingContext.Fatal(err, output)
	}
	before, _ := state.Load(home)
	mainConfig, _ := os.ReadFile(configPath)
	if len(before.ConfigEdits) != 1 || !before.ConfigEdits[0].Complete || before.ConfigEdits[0].BackupPath == "" {
		testingContext.Fatal("fixture did not create Include journal", output)
	}
	backup, _ := os.ReadFile(before.ConfigEdits[0].BackupPath)
	external.ssh = func(command app.Command) error {
		payload := command.Args[len(command.Args)-1]
		if strings.Contains(payload, "KEYFLOW_ALTERNATE") {
			fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
		} else if strings.Contains(payload, "KEYFLOW_REVOCATION_READ") {
			fmt.Fprintln(command.Stdout, "KEYFLOW_PRESENT")
		} else {
			fmt.Fprintln(command.Stdout, "KEYFLOW_REMOVED")
		}
		return nil
	}
	output, err = runCLI(testingContext, home, "4\n1\np\nyes\nyes\n6\n1\nyes\n0\n", external)
	after, loadErr := state.Load(home)
	actualMain, _ := os.ReadFile(configPath)
	actualBackup, _ := os.ReadFile(before.ConfigEdits[0].BackupPath)
	managed, _ := os.ReadFile(filepath.Join(state.Directory(home), "config"))
	if err != nil || loadErr != nil || len(after.Keys) != 0 || !reflect.DeepEqual(before.ConfigEdits, after.ConfigEdits) || string(actualMain) != string(mainConfig) || string(actualBackup) != string(backup) || string(managed) != "Host *\n" || after.Hosts[0].ConfigKeyID != "" || after.Configs[0].PendingSHA256 != "" {
		testingContext.Fatal("configuration ownership/Include history changed", err, loadErr, after, output)
	}
}

func TestLocalDeletionPreservesFilesWithChangedContentOrConfiguration(testingContext *testing.T) {
	for _, changed := range []string{"public", "private", "config", "symlink"} {
		testingContext.Run(changed, func(testingContext *testing.T) {
			home, external := prepareRegistration(testingContext, true)
			if err := state.Update(home, func(document *state.State) error {
				document.Registrations[0].Status, document.Registrations[0].Stage, document.Registrations[0].Operation = "removed", "complete", "unregister"
				return nil
			}); err != nil {
				testingContext.Fatal(err)
			}
			document, _ := state.Load(home)
			path := document.Keys[0].PublicPath
			contents := otherPublicKey
			if changed == "private" {
				path, contents = document.Keys[0].PrivatePath, "changed private"
			}
			if changed == "config" {
				path, contents = filepath.Join(state.Directory(home), "config"), "unowned configuration"
			}
			if changed == "symlink" {
				victim := writeFixture(testingContext, home, "unmanaged-key", publicKey)
				if err := os.Remove(path); err != nil {
					testingContext.Fatal(err)
				}
				if err := os.Symlink(victim, path); err != nil {
					testingContext.Skipf("fixture symlink privilege unavailable: %v", err)
				}
			} else if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
				testingContext.Fatal(err)
			}
			output, err := runCLI(testingContext, home, "6\n1\nyes\n0\n", external)
			after, loadErr := state.Load(home)
			if err != nil || loadErr != nil || len(after.Keys) != 1 || len(after.Deletions) != 0 || !strings.Contains(output, "requires review") {
				testingContext.Fatal("unsafe files accepted for deletion", err, loadErr, after, output)
			}
			for _, file := range []string{document.Keys[0].PublicPath, document.Keys[0].PrivatePath} {
				if _, err := os.Stat(file); err != nil {
					testingContext.Fatal("unsafe deletion removed a file", err)
				}
			}
		})
	}
}

func TestLocalDeletionRejectsAnUnfinishedIncludeIntent(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, false)
	if err := state.Update(home, func(document *state.State) error {
		document.Registrations[0].Status, document.Registrations[0].Stage, document.Registrations[0].Operation = "removed", "complete", "unregister"
		document.ConfigEdits = []state.ConfigEdit{{Path: filepath.Join(home, ".ssh/config"), BeforeSHA256: fmt.Sprintf("%x", sha256.Sum256(nil)), AfterSHA256: fmt.Sprintf("%x", sha256.Sum256([]byte("Include \"ssh-keyflow/config\"\n")))}}
		return nil
	}); err != nil {
		testingContext.Fatal(err)
	}
	before, _ := state.Load(home)
	output, err := runCLI(testingContext, home, "6\n1\nyes\n0\n", external)
	after, loadErr := state.Load(home)
	_, fileErr := os.Stat(before.Keys[0].PublicPath)
	if err != nil || loadErr != nil || fileErr != nil || !reflect.DeepEqual(after, before) || !strings.Contains(output, "unfinished Include") {
		testingContext.Fatal("unfinished Include intent ignored", err, loadErr, fileErr, output)
	}
}

func TestPublicOnlyDeletionRetainsItsProvenance(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, false)
	if err := state.Update(home, func(document *state.State) error {
		document.Registrations[0].Status, document.Registrations[0].Stage, document.Registrations[0].Operation = "removed", "complete", "unregister"
		return nil
	}); err != nil {
		testingContext.Fatal(err)
	}
	before, _ := state.Load(home)
	output, err := runCLI(testingContext, home, "6\n1\nyes\n0\n", external)
	after, loadErr := state.Load(home)
	_, pathErr := os.Stat(before.Keys[0].PublicPath)
	if err != nil || loadErr != nil || !os.IsNotExist(pathErr) || len(after.Keys) != 0 || len(after.DeletedKeys) != 1 || !reflect.DeepEqual(after.DeletedKeys[0].Key, before.Keys[0]) || !after.Deletions[0].PrivateRemoved {
		testingContext.Fatal("public-only deletion lost provenance", err, loadErr, pathErr, output)
	}
}

func TestLocalDeletionPreservesAReappearedPath(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, true)
	if err := state.Update(home, func(document *state.State) error {
		document.Registrations[0].Status, document.Registrations[0].Stage, document.Registrations[0].Operation = "removed", "complete", "unregister"
		document.Deletions = []state.LocalDeletion{{ID: "deletion-fixture", KeyID: document.Keys[0].ID, Stage: "pending", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), PublicSHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(publicKey))), PrivateSHA256: document.Keys[0].PrivateSHA256, PublicRemoved: true}}
		return nil
	}); err != nil {
		testingContext.Fatal(err)
	}
	before, _ := state.Load(home)
	output, err := runCLI(testingContext, home, "6\n1\nyes\n0\n", external)
	after, loadErr := state.Load(home)
	if err != nil || loadErr != nil || !reflect.DeepEqual(after, before) || !strings.Contains(output, "reappeared") {
		testingContext.Fatal("reappeared path was accepted", err, loadErr, output)
	}
	for _, path := range []string{before.Keys[0].PublicPath, before.Keys[0].PrivatePath} {
		if _, err := os.Stat(path); err != nil {
			testingContext.Fatal("reappeared path or private key removed", err)
		}
	}
}
