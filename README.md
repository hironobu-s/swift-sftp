
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

Download the archive file from [GitHub Release](https://github.com/hironobu-s/swift-sftp/releases).

```
wget https://github.com/hironobu-s/swift-sftp/releases/download/1.1.1/swift-sftp-1.1.1-linux.amd64.tgz
tar xf swift-sftp-1.1.1-linux.amd64.tgz
cd swift-sftp-1.1.1
```

## Quick Start

### Edit configuration file

You need to edit the configuration file and fill the options for OpenStack configuration.

See: [swift-sftp.conf](https://github.com/hironobu-s/swift-sftp/blob/master/misc/swift-sftp.conf)


### Public key authentication

You can also see `authorized_keys` in the configuration file.

```toml
authorized_keys = "~/.ssh/authorized_keys"
```

Default value is `~/.ssh/authorized_keys`, which means all of SSH user will be accepted to swift-sftp server.

### Starting SFTP server

Providing configuration file name with `-f` option to start SFTP server

```shell
$ ./swift-sftp server -f swift-sftp.conf
2018-01-01 00:00:00 [-]  Starting SFTP server
2018-01-01 00:00:00 [-]  Use container 'https://object-storage.tyo1.conoha.io/v1/[TENANT_ID]/[CONTAINER_NAME]
2018-01-01 00:00:00 [-]  Listen: localhost:10022
```

Also use the short name ``s`` instead of ``server``

```shell
$ swift-sftp s -c [container-name]
```

Once the server started, You can connect it through the SFTP client.

```shell
$ sftp -P 10022 -i [private_key_file] hironobu@localhost
Connected to localhost.
sftp>
```

## Usage

### Allow swift-sftp accessing from public network

Edit `bind_address` option.

(before)
```toml
bind_address = "127.0.0.1:10022"
```

(after)
```toml
bind_address = "0.0.0.0:10022"
```

### OpenStack configurations

'sftp-sftp` accepts the environment variables for OpenStack authentication to access to the container.

See: [OpenStack Docs: Authentication](https://docs.openstack.org/python-openstackclient/pike/cli/authentication.html)

```bash
export OS_USERNAME=[Username]
export OS_PASSWORD=[Password]
export OS_TENANT_NAME=[Tenant name]
export OS_AUTH_URL=[URL of Identity Endpoint]
export OS_REGION_NAME=[Region name]
```

### Password authentication

You can use Password authentication method with `--password-file` option. A password file has two fields separated by colon, username and hash value.

To create your password file with `gen-password-hash` sub-command:

```shell
$ swift-sftp gen-password-hash -f hironobu > passwd
Password:
$ cat passwd
hironobu:971ec9d21d32fe4f5fb440dc90b522aa804c663aec68c908cbea5fc790f7f15d
```

### How to build

```shell
cd $GOPATH
go get github.com/hironobu-s/swift-sftp
cd $GOPATH/src/github.com/hironobu-s/swift-sftp
make setup
make
```

## License

Copyright (c) 2018 Hironobu Saito

[Released under the MIT license](https://opensource.org/licenses/mit-license.php)
