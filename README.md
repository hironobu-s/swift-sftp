
# swift-sftp

`swift-sftp` is an SFTP server that uses [OpenStack Swift Object Storage](https://docs.openstack.org/swift/latest/) as a filesystem. 

> swift-sftp is supposed to be used for [ConoHa Object Storage](https://www.conoha.jp/en/features/)


## Features

* swift-sftp deals with a single container on Object Storage.
* You can upload and download the object through SFTP client
* swift-sftp supports not only public key authentication as the default but also password authentication.

Followings are some rescrictions by the gaps of the protocols between HTTPS and SFTP.

* Doesn't support `chmod` command
* Doesn't support any operations for directories
* It takes twice times more time than uploading the object to Object Storage directly


## Install

Download the executable file on GitHub release page.

**Mac OSX**

```shell
curl -sL https://github.com/hironobu-s/swift-sftp/releases/download/latest/swift-sftp-osx.amd64.gz | zcat > swift-sftp && chmod +x ./swift-sftp
```

**Linux(amd64)**

```shell
curl -sL https://github.com/hironobu-s/swift-sftp/releases/download/latest/swift-sftp-linux.amd64.gz | zcat > swift-sftp && chmod +x ./swift-sftp
```

**Windows(amd64)**

[swift-sftp.exe](https://github.com/hironobu-s/swift-sftp/releases/download/latest/swift-sftp.exe)

**Build manually**

```shell
cd $GOPATH
go get github.com/hironobu-s/swift-sftp
cd $GOPATH/src/github.com/hironobu-s/swift-sftp
make setup
make
```

## How to set up a server

### OpenStack configurations

'sftp-sftp` must have the environment variables for OpenStack authentication to access to the container.

See: [OpenStack Docs: Authentication](https://docs.openstack.org/python-openstackclient/pike/cli/authentication.html)

```bash
export OS_USERNAME=[Username]
export OS_PASSWORD=[Password]
export OS_TENANT_NAME=[Tenant name]
export OS_AUTH_URL=[URL of Identity Endpoint]
export OS_REGION_NAME=[Region name]
```

### Authentication methods

`swift-sftp` uses `($HOME)/.ssh/authorized_keys` file for Public Key authentication at default. All users in the list will be permitted to connect to the SFTP server.

You may also use Password authentication method with `--password-file` option. A password file has two fields separated by colon, username and hash value.

To create your password file with `gen-password-hash` sub-command:

```bash
$ swift-sftp gen-password-hash -f hironobu > passwd
Password:
$ cat passwd
hironobu:971ec9d21d32fe4f5fb440dc90b522aa804c663aec68c908cbea5fc790f7f15d
```

### Starting SFTP server

Providing your container name with `-c` option, and run SFTP server

```shell
$ swift-sftp server -c [container-name]
2018-01-01 00:00:00 [-]  Starting SFTP server
2018-01-01 00:00:00 [-]  Use container 'https://object-storage.tyo1.conoha.io/v1/[TENANT_ID]/[CONTAINER_NAME]
2018-01-01 00:00:00 [-]  Listen: localhost:10022
```

Also use the short name ``s`` instead of ``server``

```shell
$ swift-sftp s -c [container-name]
```

You might want to connect the server from the public network. The server will listen to the specific network address with ``-a``option.

```shell
$ swift-sftp s -a 0.0.0.0:10022 -c [container-name]
2018-01-01 00:00:00 [-]  Starting SFTP server
2018-01-01 00:00:00 [-]  Use container 'https://object-storage.tyo1.conoha.io/v1/[TENANT_ID]/[CONTAINER_NAME]
2018-01-01 00:00:00 [-]  Listen: 0.0.0.0:10022
```

Once the server started, You can connect it through the SFTP client.

```shell
$ sftp -P 10022 -i [private_key_file] hironobu@localhost
Connected to localhost.
sftp>
```

## Configuration file

`swift-sftp` also supports the configuration file with `-f` options. 

[sample-config.toml](misc/sample-config.toml)

## License

Copyright (c) 2018 Hironobu Saito

[Released under the MIT license](https://opensource.org/licenses/mit-license.php)
