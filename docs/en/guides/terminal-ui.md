# Using the TUI

[简体中文](../../zh-CN/guides/terminal-ui.md) · English · [日本語](../../guides/terminal-ui.md)

[Documentation index](../README.md)

## Launching and key bindings

Running the app with no arguments in a terminal, or running `ssh-keyflow tui`, opens the TUI. From Keys and Destinations you create and import keys, associate keys with destinations, and show and copy public keys and the manual setup guide. For a destination, choose a key that has already been created or imported. If you need a new key, create it first as a separate step. See the in-app help for the key bindings.

| Key | Action |
| --- | --- |
| Tab / Shift+Tab | Switch between Keys / Destinations |
| ↑ / ↓, Enter | Select, show details |
| `/` / `x` | Search / clear search |
| `c` / `i` | Create / import a key in Keys |
| `a` / `l` | Add a destination / associate a key with a destination |
| `d` / `u` | Remove the selected key or destination from management / dissociate an existing association |
| `p` / `g` | Show the public key / the deployment, removal and connection guide |
| `t` / `T` | Pin / unpin host keys for the selected destination |
| `y` | Copy the public key, or the runnable script of the selected guide step |
| ← / →, `b` / `n` | Previous / next guide step |
| `s` | Resume the selected unfinished local key creation |
| `v` / F5 | Read-only technical details and legacy history / reload the saved state |
| `?` / Esc / `q` | Help / back / quit |

Copying requires a terminal that supports OSC52. If it is not available, take the output of the CLI commands `keys public --key ID` / `hosts guide --host ID --key ID` into an editor or the OS clipboard. Keep the public key on one line, and do not paste the TUI's display wrapping as line breaks. You can scroll through long text with the arrow keys, PgUp / PgDn and Home / End. The minimum terminal size is 32 columns × 12 rows. In forms, move with Tab / Shift+Tab, continue with Ctrl+S, and cancel input with Esc / Ctrl+C.

The TUI separates navigation, the list and body, search and form input, messages, and the menu with frames. List items are separated by blank lines, and the selected item or input field and warnings or input errors are indicated with colors and symbols. Continue / Cancel / Help in forms are fixed in the bottom menu. On small terminals fewer items are shown, and the message frame is omitted when there is no notification. To disable colors, set the environment variable `NO_COLOR=1` before launching.

## Showing and copying the setup guide

The guide shows, step by step, scripts you can paste as-is into the matching terminal and run. When you press `g` and choose a destination and key, a form opens where you can review and change the local and destination OSes (linux / macos / windows). The defaults are the OS running the app and the saved destination OS. For a Windows destination, also specify the account type. Press Ctrl+S to generate the guide, choose a step with ← / → or `b` / `n`, and press `y` to copy only that script. Explanatory text and scripts for other terminals are not included in the copy. The CLI `hosts guide` shows all steps.

1. **Destination terminal**: run the public key deployment script for the selected destination OS. On Linux/macOS use the login user's POSIX shell; for a Windows standard account use that user's PowerShell; for a Windows administrator account use a PowerShell opened as administrator. The script keeps existing public keys, appends the new one and sets the required permissions. If the same public key is already present, it is not appended again.
   - **If the host keys are not pinned**: the guide next shows steps to display the host public keys on the destination (on Windows, in a PowerShell opened as administrator) and paste them into `ssh-keyflow hosts pin` on the local terminal. In the TUI, choose the destination and press `t` to do the same paste and confirmation in the native terminal. If you regenerate the guide after pinning, the SSH config append step that follows gains the pinning lines. These are shown only when the local OS is the same as the OS running the app.
2. **Local terminal**: for a key with a private key, run the SSH config append script for the selected local OS. Windows uses PowerShell; Linux/macOS use a POSIX shell. If you choose an OS different from the one running the app, specify the matching private key path as seen from that terminal. The script keeps existing content and stops if the same explicit alias exists. Check Include, wildcards, Match and precedence yourself before running it.
3. **Local terminal (key with a passphrase)**: a step that starts ssh-agent automatically. The app determines from the private key's format header whether it has a passphrase, and shows this step for keys with a passphrase or whose status cannot be determined (it says so when it cannot determine it). It adds `AddKeysToAgent yes` to the SSH config so that the key is loaded into the agent on the first connection and kept until the agent stops. On Windows, an administrator PowerShell sets the `ssh-agent` service to start automatically. On macOS, it checks the launchd agent and uses `UseKeychain yes` to store the passphrase in the keychain. On Linux, it only checks that the agent is reachable, and guides you to choose among keyring, a systemd user service or a shell profile.
4. **Local terminal**: run `ssh 別名` (別名 is the alias) from the final step.

For a public-key-only key, the guide uses the public key file as `IdentityFile` and states that the matching private key must be loaded into the SSH agent. The ssh-agent step is not shown. The guide also includes manual unregistration instructions. Make the SSH server and network available beforehand, and for a destination with a custom configuration, check the actual `AuthorizedKeysFile` / `Match`. The Windows append script handles existing files in UTF-8 and stops without rewriting files in UTF-16 or similar.

## Host key pinning

Pressing `t` in the destination list or details switches to the native terminal and shows the command to run on the destination. Paste the host public keys printed on the destination's own terminal, press Enter on an empty line, check the displayed fingerprints, and type `yes`. If different keys are already pinned, the old and new fingerprints and a warning are shown. Type `yes` only if you have confirmed the host key change on the destination itself. `T` unpins after confirmation. The details screen shows `Host keys: pinned (N)` / `not pinned` / `unknown`. `unknown` means the pin file cannot be read; the app does not repair it automatically.

An association between a key and a destination is a local note. Saving, showing or copying it does not mean registration or connection succeeded, and there is no feature for recording success after manual work. The app does not run scripts or perform SSH connections, registration or unregistration, and does not edit SSH configuration or Include directly. Public keys and local config are appended only when you run the copied scripts yourself. Delete key files manually after confirming you need to.

Next, see [Your first connection setup](../tutorials/first-connection.md) or [Using the CLI](cli.md).
