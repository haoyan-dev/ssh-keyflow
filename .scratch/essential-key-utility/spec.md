# Essential SSH key utility

Status: implemented and verified locally; awaiting human integration review
Base: 404a3ebe616799b582a9bc8438bf47360065847c

## Objective

SSH KeyFlowを、ローカルの鍵管理とコピペによる接続設定支援に絞る。
この仕様は`.scratch/ssh-onboarding/`の自動登録・解除・設定編集の要件を置き換える。

## Scope and acceptance

- Ed25519作成、公開鍵のみを含む明示的インポート、鍵ペア検証、作成中断の再開を残す。
- Keys / Destinationsの2画面を中心とするTUIと薄いCLIを残す。
- 接続先は既存の鍵を選んで追加する。新しい鍵は独立した作成操作で用意する。
- 公開鍵、OS/Windowsアカウント別の配置先・手動解除案内、SSH Hostスニペット、接続コマンドを表示・コピーできる。
- 鍵と接続先の関連付けを保存できる。関連付けやコピーは登録成功や接続成功を意味しない。
- 自動登録・解除、パスワード/ブートストラップ鍵選択、接続プローブ、SSH config/Includeの編集、ローカル鍵削除、旧番号メニュー、再帰的config探索を削除する。
- サーバー導入・起動・ファイアウォール変更用スクリプトを生成しない。
- 既存鍵、v1状態の全履歴と中断記録、SSH config/Include/バックアップのバイト列を維持する。
- v1を読み、最初の保存でv2に移行する。新しいAssociationsは履歴とは別の関連付け。旧アプリへ戻す可能性がある場合、最初の変更操作より前にv1状態ファイルを利用者が別途退避する。`.bak`は直前保存のローリングバックアップであり、複数回保存する鍵作成では同じ操作中にv2へ更新される。
- 古い削除途中の鍵は要確認として使用を拒否し、他の鍵の操作を妨げない。古いリモート操作は読み取り専用で残し、自動的に完了へ変更しない。
- 状態検証、排他ロック、直前状態のバックアップ、原子的保存、鍵の上書き防止と同一性検証を維持する。
- 全Goテスト/vet、配布・インストーラー検証、6ターゲットのビルドを実施し、実行条件と対象コミットを記録する。

## Backend interface

`app.Request`: Operation (`list`, `create`, `import`, `add`, `associate`, `guide`, `public`, `resume`), Name, PublicPath, PrivatePath, Key, Destination (= state.Host), Host, Resume.

`add` requires an existing Key and Destination. `associate` requires Host and Key.
`guide` requires Host and Key and is read-only. `public` requires Key and is read-only.
`resume` accepts only unfinished local Creation IDs.

`app.Snapshot(Options)` returns View{State, Review}; State exposes legacy records unchanged plus Associations []state.Association{HostID, KeyID}.
`app.PublicKey(home, keyRef) (string,error)` and `app.Guide(home, hostRef, keyRef) (string,error)` provide validated copyable text without writes, dependency probes or SSH processes.
`app.Run` takes explicit Request, Confirm and the existing Runner/native terminal handoff for local key creation/import.
Existing local Report types remain available for native handoff; remote reports are removed.

## Dependencies

Backend/state and CLI/TUI implementation can proceed independently against this interface.
Documentation and final standards/specification reviews follow integration.

## Completion

Source: `22275867764d74f5dc60c4fc8d48a70d141e856b`.
Local implementation and verification completed on 2026-11-08. Tickets 01–03 are ready-for-human.
See `docs/verification.md` for tested conditions, archive evidence and native execution limits.
Connection commands use the validated destination alias after the user pastes the SSH Host snippet.
Standards and Astra specification findings were resolved; no remote publication or merge is part of this task.
