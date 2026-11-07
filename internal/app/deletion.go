package app

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"ssh-keyflow/internal/state"
)

func deleteLocal(options Options) error {
	return state.WithLock(options.Home, func(store *state.LockedStore) error {
		document, err := store.Load()
		if err != nil {
			return err
		}
		for index, key := range document.Keys {
			fmt.Fprintf(options.Output, "%d: %s [%s] %s\n", index+1, key.Name, key.ID, key.Fingerprint)
		}
		selectedIndex := 0
		if options.Request != nil {
			selectedIndex, err = keyIndex(document, options.Request.Key)
			if err != nil {
				return err
			}
		} else {
			choice, err := prompt(options, "Managed local key number to delete/resume; 0: Back")
			if err != nil || choice == "0" {
				return err
			}
			selection, err := strconv.Atoi(choice)
			if err != nil || selection < 1 || selection > len(document.Keys) {
				return fmt.Errorf("invalid managed key selection")
			}
			selectedIndex = selection - 1
		}
		keyIndex := selectedIndex
		key := document.Keys[keyIndex]
		for _, registration := range document.Registrations {
			if registration.KeyID == key.ID && (registration.Status != "removed" || registration.Stage != "complete" || registration.Operation != "unregister") {
				return fmt.Errorf("recorded registration %s is not confirmed removed; unregister with alternate authentication or manual guidance; local files preserved", registration.ID)
			}
		}
		deletionIndex := -1
		for index, deletion := range document.Deletions {
			if deletion.Stage == "pending" {
				if deletion.KeyID != key.ID {
					return fmt.Errorf("requires review: resume unfinished deletion %s first", deletion.ID)
				}
				deletionIndex = index
			}
		}
		otherKeys := document
		otherKeys.Keys = append([]state.Key{}, document.Keys[:keyIndex]...)
		otherKeys.Keys = append(otherKeys.Keys, document.Keys[keyIndex+1:]...)
		otherKeys.Deletions = nil
		if err := integrity(otherKeys); err != nil {
			return err
		}
		public, err := plainConfig(key.PublicPath)
		if err != nil {
			return err
		}
		private := []byte(nil)
		if key.PrivatePath != "" {
			private, err = plainConfig(key.PrivatePath)
			if err != nil {
				return err
			}
		}
		if deletionIndex < 0 {
			identity, err := fingerprint(public)
			if err != nil || identity != key.Fingerprint || (key.PrivatePath != "" && (private == nil || configDigest(private) != key.PrivateSHA256)) {
				return fmt.Errorf("requires review: managed files missing or changed; preserved")
			}
		} else {
			deletion := document.Deletions[deletionIndex]
			if (public != nil && (deletion.PublicRemoved || configDigest(public) != deletion.PublicSHA256)) || (private != nil && (deletion.PrivateRemoved || configDigest(private) != deletion.PrivateSHA256)) {
				return fmt.Errorf("requires review: deletion paths changed/reappeared; preserved")
			}
		}
		if _, _, err := ownedConfig(options, &document); err != nil {
			return err
		}
		for _, edit := range document.ConfigEdits {
			if !edit.Complete {
				return fmt.Errorf("requires review: unfinished Include edit; preserve configuration and backup journal before local deletion")
			}
		}
		fmt.Fprintf(options.Output, "Only recorded destinations are known; keys might be registered elsewhere. Separate local deletion never revokes remote access.\nDelete %s [%s]\nFingerprint: %s\nPublic: %s\nPrivate: %s\nRetain stable IDs, registration history and deletion tombstone.\n", key.Name, key.ID, key.Fingerprint, key.PublicPath, key.PrivatePath)
		answer, err := prompt(options, "Type yes to delete only these managed local files (or resume the saved deletion)")
		if err != nil || answer != "yes" {
			return err
		}
		if deletionIndex < 0 {
			identifier, err := newID("deletion")
			if err != nil {
				return err
			}
			document.Deletions = append(document.Deletions, state.LocalDeletion{ID: identifier, KeyID: key.ID, Stage: "pending", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), PublicSHA256: configDigest(public), PrivateSHA256: key.PrivateSHA256, PrivateRemoved: key.PrivatePath == ""})
			deletionIndex = len(document.Deletions) - 1
			if err := store.Save(document); err != nil {
				return err
			}
		}
		if err := removeKeyConfiguration(options, store, &document, key.ID, ""); err != nil {
			return err
		}
		deletion := &document.Deletions[deletionIndex]
		for _, target := range []struct {
			path, digest string
			removed      *bool
		}{{key.PublicPath, deletion.PublicSHA256, &deletion.PublicRemoved}, {key.PrivatePath, deletion.PrivateSHA256, &deletion.PrivateRemoved}} {
			if target.path == "" {
				continue
			}
			current, err := plainConfig(target.path)
			if err != nil {
				return err
			}
			if current != nil {
				if *target.removed || configDigest(current) != target.digest {
					return fmt.Errorf("requires review: deletion path changed; preserved")
				}
				if err := os.Remove(target.path); err != nil {
					return err
				}
			}
			*target.removed = true
			if err := store.Save(document); err != nil {
				return err
			}
		}
		deletion.Stage = "complete"
		document.DeletedKeys = append(document.DeletedKeys, state.DeletedKey{Key: key, DeletionID: deletion.ID, DeletedAt: time.Now().UTC().Format(time.RFC3339Nano)})
		document.Keys = append(document.Keys[:keyIndex], document.Keys[keyIndex+1:]...)
		if err := store.Save(document); err != nil {
			return err
		}
		fmt.Fprintln(options.Output, "Managed local files deleted; identity tombstone and history retained")
		return nil
	})
}
