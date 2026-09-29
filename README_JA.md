# AGraphs

<img src="frontend/public/agraphs-mark.svg" width="64" alt="AGraphs" />

テキスト・画像・3D アセットをアプリに接続する AI API プラットフォーム。
GPT / Claude 互換 API と Tripo / Meshy の非同期アセット API を提供します。
顧客向け UI は AGraphs、管理画面は Sub2API を基盤としています。

- [README・セットアップ](README.md)
- [API・顧客ポータル](docs/AGRAPHS.md)
- [3D OpenAPI](docs/sup3/openapi.json)
- [上流 README（日本語）](docs/upstream/README_JA.md)
- [LGPL-3.0 ライセンス](LICENSE)

Web サイト内の `/connect` と `/docs` に接続例と API ドキュメントがあります。
モデルの利用可否はキーの権限と上流設定に依存します。

`/keys` で作成した AGraphs キーを使い、`/connect` でプロトコルと入力を選択すると、cURL / JavaScript / Python / JSON の例を取得できます。ページ内の接続確認と 3D 見積もりは生成タスクを作成しません。生成はアプリのサーバーから API を呼び出して実行します。

API 層は認証・プロトコル変換・モデルルーティング・非同期タスク・利用量・アセット配信を担当し、エディターやシーンなどの業務ロジックはアプリ層に置きます。API ページとドキュメントのコード選択色も読みやすく調整しています。
