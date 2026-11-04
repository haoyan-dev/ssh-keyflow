# SSH実装時の確認事項

確認日：2026-11-04。spec.md、requirements.md、plan.md の要件を実装する際の資料。新しい機能要件は追加しない。

## 認証と新規接続

- OpenSSHの `ControlMaster=no` だけでは既存マスターの再利用を防げない。新規認証を必要とする操作は `ControlPath=none` も指定する。
- `IdentitiesOnly=yes` は設定ファイルに列挙された他のIdentityFileも許す。指定鍵だけの検証は設定読込を分離し、IdentityAgent、CertificateFile、認証方式も明示する。
- `ssh -F none` はユーザーとシステムの設定ファイルを読まない。管理対象外のMatch execなどを実行せず直接接続する手段として使える。ユーザーが設定したProxyJump等も読まなくなるため、接続失敗時の案内で説明する。
- パスワード方式で別認証を確認するときはPubkeyAuthenticationを無効にする。別鍵方式ではパスワード等へのフォールバックを許さず、解除対象と同じ指紋の別ファイルも拒否する。
- 初回はホスト鍵を端末で確認する。変更された鍵を続行させる設定やknown_hostsの自動削除を使わない。テストのHomeを使うときも実ユーザーのknown_hostsへ書き込まない。
- アプリが入力を先読みすると、子プロセスに渡す端末入力を取り込むことがある。対話の読み取りとssh/ssh-keygenの端末入力を分離する。

資料：

- https://man.openbsd.org/ssh.1 （-F、-i、VERIFYING HOST KEYS）
- https://man.openbsd.org/ssh_config.5 （ControlMaster、ControlPath、IdentitiesOnly、IdentityAgent、IdentityFile、CertificateFile）

## 公開鍵ファイル

- 登録・解除は鍵のtypeとbase64 blobで同一鍵を判断し、コメントだけの差を別鍵としない。鍵オプション内の文字列を誤一致させない。
- 既存の他の鍵、コメント、行末を保持する。登録先・所有者・権限・シンボリックリンク等を確認できない場合の手動経路を用意する。
- 結果不明の操作は読み取り照合してから再開する。接続エラーから登録の成否を推測しない。
- 自動処理と手動コピペ手順は同じ処理から生成する。公開鍵のコメントやパスを無加工でシェルコードへ連結しない。

## Windows

- 通常ユーザーと管理者グループでは標準の登録先が異なる。管理者用ファイルは複数ユーザーに影響するため、実際の対象と共有範囲の承認が必要。
- SID S-1-5-32-544はAdministrators、S-1-5-18はSYSTEM。ローカライズされた表示名に依存しない。既存ACLを無断でリセットしない。
- Windows PowerShell 5.1にも対応する処理にする。既存ファイルのBOMやCRLF、登録処理の文字エンコーディングを確認する。

資料：

- https://learn.microsoft.com/en-us/windows-server/administration/openssh/openssh_keymanagement
- https://learn.microsoft.com/en-us/windows-server/administration/openssh/openssh-server-configuration

## 実行確認の区分

生成したコマンドのテストは、実SSHサーバーでの権限や設定の動作確認を代替しない。Git for WindowsのBashとPowerShellでの一時ファイル実行は、Linux/macOS/WindowsサーバーのネイティブSSH結合検証とは分けて記録する。
