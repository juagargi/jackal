# jackal

An XMPP server written in Go.

This repository is a fork of [ortuman/jackal](https://github.com/ortuman/jackal) making it available for SCION/QUIC. Refer to the original repository for general usage.

## About

jackal is a free, open-source, high performance XMPP server which aims to be known for its stability, simple configuration and low resource consumption.

## Features

jackal supports the following features:

- Customizable
- Enforced SSL/TLS
- Stream compression (zlib)
- Database connectivity for storing offline messages and user settings ([BadgerDB](https://github.com/dgraph-io/badger), MySQL 5.7+, MariaDB 10.2+, PostgreSQL 9.5+)
- Cross-platform (OS X, Linux)

## Installing

### Getting Started

To start using jackal, install Go 1.13+ and run the following commands:

```bash
$ go get -d github.com/ortuman/jackal
$ cd $GOPATH/src/github.com/ortuman/jackal
$ make install
```

This will retrieve the code and install the `jackal` server application into your `$GOPATH/bin` path.

By default the application will try to read server configuration from `/etc/jackal/jackal.yml` file, but alternatively you can specify a custom configuration path from command line.

```sh
$ jackal --config=$GOPATH/src/github.com/ortuman/jackal/example.jackal.yml
```

### MySQL database creation

Grant right to a dedicated 'jackal' user (replace `password` with your desired password).

```sh
echo "CREATE USER IF NOT EXISTS 'jackal'@'localhost' IDENTIFIED BY 'password';" | mysql -h localhost -u root -p
echo "GRANT ALL ON jackal.* TO 'jackal'@'localhost';" | mysql -h localhost -u root -p
```

Create 'jackal' database (using previously created password).

```sh
echo "CREATE DATABASE jackal;" | mysql -h localhost -u jackal -p
```

Download lastest version of the [MySQL schema](sql/mysql.up.sql) from jackal Github repository.

```sh
wget https://raw.githubusercontent.com/ortuman/jackal/master/sql/mysql.up.sql
```

Load database schema into the database.

```sh
mysql -h localhost -D jackal -u jackal -p < mysql.up.sql
```

Your database is now ready to connect with jackal.

### Using PostgreSQL

Create a user and a database for that user:

```sql
CREATE ROLE jackal WITH LOGIN PASSWORD 'password';
CREATE DATABASE jackal;
GRANT ALL PRIVILEGES ON DATABASE jackal TO jackal;
```

Run the postgres script file to create database schema. In jackal's root directory run:

```sh
psql --user jackal --password -f sql/postgres.up.psql
```

Configure jackal to use PostgreSQL by editing the configuration file:

```yaml
storage:
  type: pgsql
  pgsql:
    host: 127.0.0.1:5432
    user: jackal
    password: password
    database: jackal
```

That's it!

## Push notifications

Support for [XEP-0357: Push Notifications](https://xmpp.org/extensions/xep-0357.html) is not yet available in `jackal`.

However there's a chance to forward offline messages to some external service by configuring offline module as follows:

```yaml
  mod_offline:
    queue_size: 2500
    gateway:
      type: http
      auth: a-secret-token-here
      pass: http://127.0.0.1:6666
```

Each time a message is sent to an offline user a `POST` http request to the `pass` URL is made, using the specified `Authorization` header and including the message stanza into the request body.

## Run jackal in Docker

Set up `jackal` in the cloud in under 5 minutes with zero knowledge of Golang or Linux shell using our [jackal Docker image](https://hub.docker.com/r/ortuman/jackal/).

```bash
$ docker pull ortuman/jackal
$ docker run --name jackal -p 5222:5222 ortuman/jackal
```

## Supported Specifications
- [RFC 6120: XMPP CORE](https://xmpp.org/rfcs/rfc6120.html)
- [RFC 6121: XMPP IM](https://xmpp.org/rfcs/rfc6121.html)
- [RFC 7395: XMPP Subprotocol for WebSocket](https://tools.ietf.org/html/rfc7395)
- [XEP-0004: Data Forms](https://xmpp.org/extensions/xep-0004.html) *2.9*
- [XEP-0012: Last Activity](https://xmpp.org/extensions/xep-0012.html) *2.0*
- [XEP-0030: Service Discovery](https://xmpp.org/extensions/xep-0030.html) *2.5rc3*
- [XEP-0049: Private XML Storage](https://xmpp.org/extensions/xep-0049.html) *1.2*
- [XEP-0054: vcard-temp](https://xmpp.org/extensions/xep-0054.html) *1.2*
- [XEP-0077: In-Band Registration](https://xmpp.org/extensions/xep-0077.html) *2.4*
- [XEP-0092: Software Version](https://xmpp.org/extensions/xep-0092.html) *1.1*
- [XEP-0138: Stream Compression](https://xmpp.org/extensions/xep-0138.html) *2.0*
- [XEP-0160: Best Practices for Handling Offline Messages](https://xmpp.org/extensions/xep-0160.html) *1.0.1*
- [XEP-0163: Personal Eventing Protocol](https://xmpp.org/extensions/xep-0163.html) *1.2.1*
- [XEP-0191: Blocking Command](https://xmpp.org/extensions/xep-0191.html) *1.3*
- [XEP-0199: XMPP Ping](https://xmpp.org/extensions/xep-0199.html) *2.0*
- [XEP-0220: Server Dialback](https://xmpp.org/extensions/xep-0220.html) *1.1.1*
- [XEP-0237: Roster Versioning](https://xmpp.org/extensions/xep-0237.html) *1.3*

## Join and Contribute

The [jackal developer community](https://gitter.im/jackal-im/jackal?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=readme.md) is vital to improving jackal future releases.  

Contributions of all kinds are welcome: reporting issues, updating documentation, fixing bugs, improving unit tests, sharing ideas, and any other tips that may help the jackal community.

## Code of Conduct

Help us keep jackal open and inclusive. Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md).

## Licensing

jackal is licensed under the GNU General Public License, Version 3.0. See
[LICENSE](https://github.com/ortuman/jackal/blob/master/LICENSE) for the full
license text.

## Contact

If you have any suggestion or question:

Miguel Ángel Ortuño, JID: ortuman@jackal.im, email: <ortuman@pm.me>


# SCION Specifics (new in this fork)

### SCION config

In .yml configuration file, you can provide different options for the incoming SCION connections:
* addr: SCION address where the server is listening for incoming connections (if "localhost", jackal determines the SCION localhost on startup)
* port: Port listening for incoming SCION connections (default is 52690)
* keep_alive: Time after which the unresponsive connection is broken.
* cert_path: Absolute path to the certificate used in creating QUIC connection (can be the same as the certificate used in c2s IP connection)
* priv_key_path: Private key corresponding to the certificate, also used in creating QUIC connection

### Install MySQL

```shell
sudo apt-get install mysql-server
mysql_secure_installation
```

### Name resolution
In the router/hosts section of the .yml file, replace the "name" entry with your server name. Make sure that this name resolves over DNS to a valid IP address where the server will be accepting client requests. Also, specify the paths to the TLS private key and certificate in the tls section, as well as in the scion_transport section.
Finally, your hostname specified in the router/hosts section needs to resolve to a valid SCION address on the specified [RAINS](https://github.com/netsec-ethz/rains) server. SCION address where the RAINS server is running needs to be specified in the config file at ~/go/src/github.com/scionproto/scion/gen/rains.cfg. Simply put the address of the RAINS server together with the port inside this file.
NOTE: when building jackal, both SCION and IP address mappings from /etc/hosts file will be loaded.

## Connect to jackal
Once you have the config file setup correctly, together with the MySQL database, valid certificates and the DNS and RAINS entries, you can run your jackal server. 
```shell
./jackal -c example.jackal.yml
```

## Notes for local testing
To be used only when playing with jackal.

### User registration
Since some XMPP clients do not support in-band registration (e.g. Profanity), users need to be created manually. So far, the only way is to manually add them into the MySQL database created previously. For example:

```shell
mysql -h localhost -u jackal -p
use jackal;
insert into users (`username`, `password`, `last_presence`, `last_presence_at`, `updated_at`, `created_at`) values ('user1', 'asdf', '<presence from="user1@localhost/profanity" to="user1@localhost" type="unavailable"/>', '2019-04-19 18:42:58', '2019-04-19 18:42:58', '2019-04-19 18:42:58');
```

### Generating self-signed certificates
If you need to create self-signed certificates, you might find this [post](https://stackoverflow.com/questions/21488845/how-can-i-generate-a-self-signed-certificate-with-subjectaltname-using-openssl) useful.
