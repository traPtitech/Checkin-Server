# Checkin (仮称)

## 概要

最初のリリースは以下の機能を目的にする

- Stripe での入部費/部費の支払いの受付
- Stripe における入出金の一覧

また、サークルが Stripe に依存せず、いつでも切り替えられることを原則にする
-> Stripe をオフにして口座振り込みを指示できるようにする

## 処理のフロー

### 入部費の支払いの受付

部員でなくても申請した人は誰でも使えるようにする

1. 大学のメールアドレスの所有者確認を行う
   - メールを送信する
   - この時に入部希望者がメールアドレスを所有していることを確認したい
     - 請求書が大学メアドに届く
2. 請求書を発行する
   - <https://docs.stripe.com/invoicing/integration>
   - 商品はすでにあるものを使う想定
     - 前期は 4000、後期は 2000
   - [Customer](https://docs.stripe.com/api/customers) オブジェクトを作成する
     - リクエストされたメールアドレスの Customer オブジェクトがすでにあれば、Customer オブジェクトは作成しない
   - 請求書を発行する
3. 支払われたら会計に通知
   - Webhook で [invoice.paid](https://docs.stripe.com/api/events/types?event_types-invoice.paid) イベントを受け取る

### 部費の支払いの受付

公開にはしないけど、再入部にも使いたいから traQ アカウントが凍結されてても使えるようにする

1. ログインまたは traQ アカウントの確認を行う
   - アカウントが凍結されている場合は traQ ID を入力してもらう、この際 ID が存在するものか確認する
   - ID がわからない場合は役員対応に誘導する
2. ユーザーのリクエストで請求書を発行する
   - metadataにtraQID
   - <https://docs.stripe.com/invoicing/integration>
   - 商品はすでにあるものを使う想定
     - 後期入部は2000、前期は4000 で入部費を機械的に決定
   - [Customer オブジェクト](https://docs.stripe.com/api/customers) を作成する
     - DB に メールのハッシュ値に紐づいた Customer オブジェクトが保存されていれば、Customer オブジェクトは作成しない
     - DB にない場合でも、リクエストされたメールアドレスの Customer オブジェクトがすでにあれば、Customer オブジェクトは作成せず、DB に保存する
     - Stripe の Customer IDと メールのハッシュ値を DB に保存
   - 請求書を発行する
3. アカウントが凍結されていれば、支払われたら会計に通知
   - Webhook で [invoice.paid](https://docs.stripe.com/api/events/types?event_types-invoice.paid) イベントを受け取る

### 入出金の一覧

会計のみ
invoice と checkout.sessions を一覧

## API 設計

TODO: 一部の API を管理者のみにする

### verify-email
- isct アドレスを受け取る
- sendgrid?でメールを送信して isct アドレスの所有者であることを確認
- JWT(isct mail address)を発行してクライアントで持っとく
- リダイレクト先をクエリパラメータに持つ
- リクエストにつけて送る


### Customer

[Customer オブジェクト](https://docs.stripe.com/api/customers): 請求書を受け取るユーザー

- 常にメールアドレスのハッシュ値でサービス内はID管理(isct限定)
- メールアドレス自体はstripeに保存

- traQ ID に紐づいていない Customer オブジェクトがすでにある場合

Customer の name or metadata に traQID 入れるとかがログは見やすそう

#### GET

以下のいずれかのパラメーターを受け取り、該当する Customer オブジェクトを返す
自分のもののみ

- Customer ID
- traQ ID
- メールアドレス(from JWT)

#### POST or PUT

以下のパラメーターを受け取り、Customer オブジェクトを作成して返す
自分のもののみ

- メールアドレス(from JWT)
- 名前(カタカナ)
- traQ ID (任意)

#### PATCH

以下のパラメーターを受け取り、Customer オブジェクトを更新して返す
自分のもののみ

- メールアドレス(from JWT)
- 名前(カタカナ)
- traQ ID (任意)

### Invoice

[Invoice オブジェクト](https://docs.stripe.com/api/invoices): 請求書

#### POST

以下のパラメーターを受け取り、Invoice オブジェクトを作成して返す
自分のもののみ

- [ドキュメント](https://docs.stripe.com/invoicing/integration#create-invoice-code)に書かれてる手順が多くてどのパラメーターが必要かわからない
- mailadress (from JWT)
- 多分 Product ID
  - ダッシュボードからは設定できるから多分できるけど、API ドキュメントに見当たらなかったから不明
  - Price ID を使うかも

:@kaitoyama:作成手順は以下の通り

1. customer と期限を使って請求書(`invoice`)を作る
2. customer と商品(`price`)、請求書(`invoice`)をつかって請求書アイテム(`invoice_item`)を作る
3. 請求書を確定させる
   - invoice を送信(`send`)する(customer のメアドに飛ぶ)
   - finalize する(event の中に hosted_url があるので、それを使うと支払いのページに飛ぶ)

### InvoicePaid

Webhook の [invoice.paid](https://docs.stripe.com/api/events/types?event_types-invoice.paid) イベントを受け取って会計に通知する

- <https://dashboard.stripe.com/webhooks> で Webhook を登録する

### Invoices

#### GET

請求書由来の入金一覧
会計のみ
<https://docs.stripe.com/api/invoices/list> がよさそうに見えるけど、stateでの絞り込みがないのが厳しいかも

- id
- 金額
- 日時
- customer への参照、または customer の traQ ID などのデータ
- 支払い状況
- 支払いのid (dashboard への url を生成するかも)
- 商品に関する情報または商品への参照
- 他にも必要そうなのあれば

Stripe の API をベースに filter 系のクエリパラメータもほしい

##### checkout_sessions

オンライン決済ページ由来の入金一覧

<https://docs.stripe.com/api/checkout/sessions/list> がよさそう

- id
- 金額
- 日時
- traQ IDとかメアドとか氏名とかのカスタムフィールド
- 支払い状況
- 支払いのid (dashboard への url を生成するかも)
- 商品に関する情報または商品への参照
- 他にも必要そうなのあれば

filter 系のクエリパラメータも

##### transfers

このサービスで Jomon の支払いを管理するなら必要だけどまだいい
<https://docs.stripe.com/api/transfers/list>

## DB 設計

- 管理者(会計) `deprecated`
    - (traQのグループ依存か)環境変数で良いはず
- Users
  - id
  - mail_hash
  - stripe_Customer_ID

ほかいらなさそう

## UI 設計

ほぼ入部フォームと同等の機能持てそうだからもうこれが入部フォームでもいいかも

### 処理フロー

#### 入部の部費支払い

- isct アドレスを入力してもらい、メールを送信
- メールの URL から請求書の送信に必要な情報を確認または入力してもらう
  - isct アドレス
  - 名前
- Customer オブジェクトを作成
- 請求書を送信

#### 再入部の部費支払い

- isct アドレスを入力してもらい、メールを送信
- メールの URL から請求書の送信に必要な情報を確認または入力してもらう
  - isct アドレス
  - traQ ID
    - 入力値が存在しなければアラートを表示
    - わからない場合は役員対応に誘導
  - 名前
- Customer オブジェクトを作成または更新
- 請求書を送信

#### 現役部員の部費支払い

- ログインしていなければログイン
- traQ ID に紐づいた Customer オブジェクトがない場合
  - isct アドレスを入力し、メールを送信
  - メールの URL から情報の入力へ
- 請求書の送信に必要な情報の入力または確認
  - isct アドレス
  - traQ ID
  - 名前
- Customer オブジェクトを作成または更新
- 請求書を送信

### ページ

#### /

- ログインしていない場合、以下へのリンクを表示
  - `/membership`
  - `/login`
- ログインしている場合、以下へのリンクを表示
  - `/membership`
  - 管理者の場合は以下も表示
    - `/admins`
    - `/payments`

#### /login

- traQ OAuth でログイン
- リダイレクト先をクエリパラメータに持つ

#### /verify-email

- isct アドレスを入力してもらう
- メールを送信して isct アドレスの所有者であることを確認
- isct アドレスをクッキー的なのに保存
- リダイレクト先をクエリパラメータに持つ
- 細かい実装が曖昧

#### /membership

##### 1. アクセス時

- ログインしている場合
  - traQ ID に紐づいた Customer オブジェクトがない場合
    - `/verify-email?redirect=/membership` へ
  - traQ ID に紐づいた Customer オブジェクトがあるか、isct アドレスを入力済みの場合
    - 現役部員設定で請求書情報入力へ
- ログインしていない場合、新規入部か再入部か現役部員かを選択してもらう
  - 新規入部
    - isct アドレスが未入力の場合
      - `/verify-email?redirect=/membership` へ
    - isct アドレスが入力済みの場合
      - 新規入部設定で請求書情報入力へ
  - 再入部
    - isct アドレスが未入力の場合
      - `/verify-email?redirect=/membership` へ
    - isct アドレスが入力済みの場合
      - 再入部設定で請求書情報入力へ
  - 現役部員の場合
    - `/login?redirect=/membership` へ

##### 2. 請求書情報入力

- この時点で以下の状態の想定
  - isct アドレスが確定
  - 入部、再入部、現役部員のどれかに設定されている
    - クエリパラメーターかクッキー的なものか悩み
  - 現役部員であればログインしている
  - 上記でない場合は最初からやり直し
- 再入部であり、isct アドレスに対応する Customer オブジェクトがないか、Customer オブジェクトに traQ ID がない場合
  - traQ ID を入力してもらう
    - 入力値が存在しない traQ ID であればアラートを表示
    - わからない場合は役員対応に誘導
- ログインしている traQ ID や isct アドレスに対応する Customer オブジェクトがない場合
  - 名前を入力してもらう

##### 3. 請求書の送信

- 請求書の送信に必要な情報を確認してもらう
  - isct アドレス
  - traQ ID
    - 現役部員または再入部の場合のみ
  - 名前
- 確定であれば請求書を送信する
  - Customer オブジェクトを作成または更新する
    - 再入部時に入力された traQ ID は使用しない

##### 再入部時の traQ ID の入力に関するバグの可能性について

- 再入部時の traQ ID の入力は正しいとは限らない
- 再入部時の traQ ID の入力で、入力者のものでないが存在する ID が入力された場合、traQ ID の入力ミスに気付けない
- 凍結されている時に traQ ID の所有者確認ができないのが問題
  - 所有者確認ができれば問題はない
- そのため、再入部時の traQ ID はアカウント復旧の対象の設定のみに使い、 Customer オブジェクトの参照での使用や Customer オブジェクトへの traQ ID の保存は行わない

#### /admins

- 管理者の一覧
- 管理者のみ
- 登録と削除

#### /payments

- 入出金の一覧
- 管理者のみ
- フィルター
- ページネーション

### ヘッダー

- サービスロゴ
- ログインしていればアイコン
- 可能ならヘルプアイコンまたはお問い合わせリンク

## TODO

- サンドボックス環境のアクセス権を渡す
  - 本番環境に影響なく実験できる

## 余裕があったら

- 部費支払いフローにステップバーを表示
- 入部・再入部フォーム機能
  - 今 Google フォームでやってるやつ
- 部員への支払い機能
  - Jomon の払い戻し
- 部員に任意の額の請求書を送る機能
  - 部内物販を想定


## 返金周りのつらさ
0. jomonの承認された後
1. jomonから今回の返金対象と金額を抜く(これはスクリプト)
2. 会計のspread sheetで↓の対応を得る(が、これのアカウントの有無は手作業で更新)
3. connectedの仕分け(作成済み/依頼したがまだ/初めて)
4. connected新規作成
5. アカウントを見つける(本名:メアドしか今登録されていない)
6. 金額を入力(手打ち)
7. 手作業でjomonの更新