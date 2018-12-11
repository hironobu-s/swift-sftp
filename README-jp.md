
# swift-sftp

`swift-sftp`は[OpenStack Swift オブジェクトストレージ](https://docs.openstack.org/swift/latest/)(以下Swift)をバックエンドに利用するSFTPサーバーです。一般的なSFTPクライアント(WinSCPやFilezillaなど)を用いて、Swiftを操作できるようになります。

> 現在、swift-sftpは[ConoHaオブジェクトストレージ](https://www.conoha.jp/objectstorage/)で使うことを想定して開発されています。

## 機能

* 一つのswift-sftpでSwift上の一コンテナを扱います
* オブジェクトのアップロード、ダウンロードをSFTPクライアントを通じて行えます
* SFTPサーバーは公開鍵認証とパスワード認証をサポートしています

また、オブジェクトストレージ(HTTPS)とSFTPのプロトコルの違いにより以下の制約事項があります。

* パーミッションの変更(chmod)はできません
* ディレクトリはサポートしていません
* SFTPクライアントからアップロードしたオブジェクトは、一度swift-sftpが動いているサーバーにアップロードされ、その後Swiftにアップロードされます。そのためアップロードには通常の2倍の時間が必要になります。

## インストール

[GitHub Release](https://github.com/hironobu-s/swift-sftp/releases)から実行ファイルをダウンロードして展開してください。

```
wget https://github.com/hironobu-s/swift-sftp/releases/download/1.1.2/swift-sftp-1.1.2-linux.amd64.tgz
tar xf swift-sftp-1.1.2-linux.amd64.tgz
cd swift-sftp-1.1.2
```

## 使い方

### 設定ファイルの編集

ダウンロードしたファイルを展開して、設定ファイル `swift-sftp.conf` を編集します。基本的にデフォルトのままでも動作しますが、OpenStack configurations の部分は環境に合わせて設定する必要があります。

解説は設定ファイル中の[コメント](https://github.com/hironobu-s/swift-sftp/blob/master/misc/swift-sftp.conf)を見てください。

### 公開鍵認証ファイルの準備

次に、swift-sftpのSFTPサーバーにアクセス可能なクライアントを設定します。設定ファイル中の`authorized_keys`です。

```toml
authorized_keys = "~/.ssh/authorized_keys"
```

デフォルト値は`~/.ssh/authorized_keys`なので、SSHで認証可能なユーザーはそのままswift-sftpでもアクセス可能になります。必要に応じて変更してください。


### SFTPサーバーの起動

swift-sftpコマンドに`-f`オプションで設定ファイル名を渡して起動します。

```shell
$ ./swift-sftp server -f swift-sftp.conf
2018-01-01 00:00:00 [-]  Starting SFTP server
2018-01-01 00:00:00 [-]  Use container 'https://object-storage.tyo1.conoha.io/v1/[TENANT_ID]/[CONTAINER_NAME]
2018-01-01 00:00:00 [-]  Listen: localhost:10022
```

`server` は `s` に省略できます。

```shell
$ ./swift-sftp s -f swift-sftp.conf
```

サーバーが起動したら、SFTPクライアントから接続します。

```shell
$ sftp -P 10022 -i [private_key_file] hironobu@localhost
Connected to localhost.
sftp>
```

## 便利な使い方

### 外部からの接続を受け付ける

設定ファイル中の`bind_address`を変更します。

(変更前)
```toml
bind_address = "127.0.0.1:10022"
```

(変更後)
```toml
bind_address = "0.0.0.0:10022"
```

### OpenStack認証について

OpenStackの認証情報は、swift-sftpがSwiftへアクセスするために使われます。設定ファイルでも指定することはできますが、環境変数で渡すこともできます。他のOpenStack CLIツールと一緒に使う場合に便利です。

参考: [OpenStack Docs: Authentication](https://docs.openstack.org/python-openstackclient/pike/cli/authentication.html)

```bash
export OS_USERNAME=[APIユーザ名]
export OS_PASSWORD=[APIパスワード]
export OS_TENANT_NAME=[テナント名]
export OS_AUTH_URL=[Identity EndpointのURL]
export OS_REGION_NAME=[リージョン]
```

### パスワード認証を使う

パスワード認証をする場合は、設定ファイル中の`password_file`を変更して、パスワードファイルを指定してください。デフォルトでは無効です。

パスワードファイルはユーザー名とハッシュを`:`で区切ったファイルです。ハッシュ値の生成は`gen-password-hash`サブコマンドを利用すると便利です。

ハッシュ値生成とパスワードファイルの作成(ユーザー名:hironobu の場合)

```bash
$ swift-sftp gen-password-hash -f hironobu > passwd
Password:
$ cat passwd
hironobu:971ec9d21d32fe4f5fb440dc90b522aa804c663aec68c908cbea5fc790f7f15d
```

### コマンドラインオプションを使う

設定ファイルを使わず、コマンドラインオプションのみで運用することもできます。`-h`を付けるとヘルプが出ます。

全体のオプションを見る場合
```shell
./swift-sftp -h
```

サーバーのオプションを見る場合
```shell
./swift-sftp server -h
```

### 自分でビルドする

`go get` してmakeするだけです。

```shell
cd $GOPATH
go get github.com/hironobu-s/swift-sftp
cd $GOPATH/src/github.com/hironobu-s/swift-sftp
make setup
make
```

## ライセンス

Copyright (c) 2018 Hironobu Saito

[Released under the MIT license](https://opensource.org/licenses/mit-license.php)
