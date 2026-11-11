# ADR 0005: パスフレーズ付き鍵にssh-agentの自動起動ステップを加える

- 状態: 採用
- 日付: 2026-11-11
- 仕様: `.scratch/ssh-agent-autostart/spec.md`

## 背景

パスフレーズ付きの鍵では接続のたびに入力が必要になる。ssh-agentを
自動起動し、最初の接続で鍵を読み込めば以後の入力を省ける。
ADR 0002はサーバー設定を除外し、ローカルではSSH configへの追記だけを
スクリプト化している。

## 決定

管理済み鍵ペアの秘密鍵ファイルの形式ヘッダーだけを読み、暗号名
（OpenSSH形式の cipher、PEMの `Proc-Type: 4,ENCRYPTED`、
`ENCRYPTED PRIVATE KEY`）でパスフレーズの有無を判定する。
復号・ssh-keygenの起動・入力要求は行わず、結果を保存しない。
パスフレーズなしと判定した鍵ではガイドを変えない。

パスフレーズ付き、または判定できない鍵では、ローカルのHostブロックに
`AddKeysToAgent yes` を加え、config追記と接続の間にssh-agentステップを置く。
読み込んだ鍵はagentが止まるまで保持する（有効期限は付けない）。
判定できない場合は理由と「パスフレーズ付きの鍵でのみ必要」と明示する。
ADR 0003に従い、別端末向けに指定された秘密鍵パスは検査せず判定不能とする。
公開鍵のみの鍵にはローカルのステップがないため追加しない。

ADR 0002のローカル側スクリプトの範囲を、ssh-agentについて次のとおり広げる。

- Windows：管理者のPowerShellで `ssh-agent` サービスを自動起動に設定し起動する。
  設定済みなら変更しない。PATH上のsshがWindows OpenSSHでなければ警告する。
- macOS：launchdが起動するagentへの到達だけを確認し、configに
  `IgnoreUnknown UseKeychain` と `UseKeychain yes` を加える。
- Linux：`SSH_AUTH_SOCK` のagentへの到達だけを確認し、デスクトップのkeyring、
  systemdユーザーサービス、シェルプロファイルの選択肢を説明する。
  ユニットの導入やプロファイルの編集は行わない。

接続先のSSHサーバーの導入・起動は引き続き扱わない。アプリはスクリプトの
実行・agentの起動・パスフレーズの受け取りを行わない。状態スキーマは変えない。
