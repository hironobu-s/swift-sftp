
# swift-sftp

`swift-sftp`は[OpenStack Swift オブジェクトストレージ](https://docs.openstack.org/swift/latest/)(以下Swift)をバックエンドに利用するSFTPサーバーです。一般的なSFTPクライアント(WinSCPやFilezillaなど)を用いて、Swiftを操作できるようになります。

> 現在、swift-sftpは[ConoHaオブジェクトストレージ](https://www.conoha.jp/objectstorage/)で使うことを想定して開発されています。

## 機能

* 一つのswift-sftpでSwift上の一つのコンテナを扱います
* オブジェクトのアップロード、ダウンロードをSFTPクライアントを通じて行えます
* SFTPサーバーは公開鍵認証とパスワード認証をサポートしています。

また、オブジェクトストレージとSFTPの機能の違いにより以下の制約事項があります。

* パーミッションの変更(chmod)はできません
* ディレクトリはサポートしていません
* SFTPクライアントからアップロードしたオブジェクトは、一度sftp-swiftが動いているサーバーにアップロードされ、その後Swiftにアップロードされます。そのためアップロードには通常の2倍の時間が必要になります。

## インストール

GitHub Releaseのページから実行ファイルをダウンロードしてください。

**Mac OSX**

```shell
curl -sL https://github.com/hironobu-s/swift-sftp/releases/download/current/swift-sftp.amd64.gz | zcat > swift-sftp && chmod +x ./swift-sftp
```

**Linux(amd64)**

```shell
curl -sL https://github.com/hironobu-s/swift-sftp/releases/download/current/swift-sftp-linux.amd64.gz | zcat > swift-sftp && chmod +x ./swift-sftp
```

**Windows(amd64)**

[ZIP file](https://github.com/hironobu-s/swift-sftp/releases/download/current/swift-sftp.amd64.zip)


## 使い方

### OpenStack認証

まずOpenStackの認証情報を設定します。これらの認証情報はswift-sftpがSwiftへアクセスするために使われます。認証情報は環境変数で渡す必要があります。

参考: [OpenStack Docs: Authentication](https://docs.openstack.org/python-openstackclient/pike/cli/authentication.html)

```bash
export OS_USERNAME=[APIユーザ名]
export OS_PASSWORD=[APIパスワード]
export OS_TENANT_NAME=[テナント名]
export OS_AUTH_URL=[Identity EndpointのURL]
export OS_REGION_NAME=[リージョン]
```

### swift-sftpの認証

次にswift-sftpのSFTPサーバーにアクセス可能なクライアントを設定します。

デフォルトで`($HOME)/.ssh/authorized_keys`ファイルが読み込まれるので、ここに記述されているユーザーは公開鍵認証でアクセスできます。ファイル名は`-k`オプションで変更することもできます。

パスワード認証も利用することができ、`--password-file`オプションでパスワードファイルを指定します。パスワードファイルはユーザー名とハッシュを`:`で区切ったファイルです。ハッシュ値の生成は`gen-password`サブコマンドを利用します。

ハッシュ値生成とパスワードファイルの作成(ユーザー名:hironobu の場合)

```bash
$ swift-sftp gen-password-hash -f hironobu > passwd
Password:
$ cat passwd
hironobu:971ec9d21d32fe4f5fb440dc90b522aa804c663aec68c908cbea5fc790f7f15d
```

なお、`--password-file`オプションを指定しなかった場合、パスワード認証は無効になります。

### SFTPサーバーの起動

swift-sftpコマンドの引数にコンテナ名を渡して実行するとSFTPサーバーが起動します。

```shell
$ swift-sftp server [container-name]
2018-01-01 00:00:00 [-]  Starting SFTP server
2018-01-01 00:00:00 [-]  Use container 'https://object-storage.tyo1.conoha.io/v1/[TENANT_ID]/[CONTAINER_NAME]
2018-01-01 00:00:00 [-]  Listen: localhost:10022
```

`server` は `s` に省略できます。

```shell
$ swift-sftp s [container-name]
```

外部からの接続を受け付けるには`-a`オプションを使います。

```shell
$ swift-sftp s -a 0.0.0.0:10022 [container-name]
2018-01-01 00:00:00 [-]  Starting SFTP server
2018-01-01 00:00:00 [-]  Use container 'https://object-storage.tyo1.conoha.io/v1/[TENANT_ID]/[CONTAINER_NAME]
2018-01-01 00:00:00 [-]  Listen: 0.0.0.0:10022
```


サーバーが起動したら、SFTPクライアントから接続します。

```shell
$ sftp -P 10022 -i [private_key_file] hironobu@localhost
Connected to localhost.
sftp>
```

## ライセンス

MIT
