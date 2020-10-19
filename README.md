# Web Trigger

Web Trigger is a Go service that listens for authenticated requests to trigger server commands on demand.

## Get started

Copy the binary to a server folder like `/usr/local/bin`.

### Service definition

Create a config file like the following and adapt it to suit your tasks, tokens and commands.

```yaml
port: 5000
triggers:
  - id: my-action-1
    token: my-token-1
    command: /home/brickpop/deploy-prod.sh --param-1
    timeout: 20 # seconds
  - id: my-action-2
    token: my-token-2
    command: /home/brickpop/deploy-dev.sh "CLI arguments go here"
  # ...
```

Create the scripts for your triggers and make sure that they are executable.

### Start the service

Start the service:

```sh
$ webtrigger my-config.yaml
```

### Call a URL

Following the example config from above:

#### Trigger the task

Trigger a task by performing a `POST` request to its path with the `Authorization` header including the appropriate token.

```sh
$ curl -X POST -H "Authorization: Bearer my-token-1" http://localhost:5000/my-action-1
OK
```

```sh
$ curl -X POST -H "Authorization: Bearer my-token-2" http://localhost:5000/my-action-2
OK
```

```sh
$ curl -X POST -H "Authorization: Bearer bad-token" http://localhost:5000/my-action-2
Invalid token
```

```sh
$ curl -X POST -H "Authorization: Bearer my-token-2" http://localhost:5000/does-not-exist
Not found
```

**Note**: invoking a task that is already running will wait to start it again until the current execution has completed

#### Get the task status

A task can be in 4 different states:

```sh
$ curl -H "Authorization: Bearer my-token-1" http://localhost:5000/my-action-1
{"id":"my-action-1","status":"unstarted"}
```

```sh
$ curl -H "Authorization: Bearer my-token-1" http://localhost:5000/my-action-1
{"id":"my-action-1","status":"running"}
```

```sh
$ curl -H "Authorization: Bearer my-token-1" http://localhost:5000/my-action-1
{"id":"my-action-1","status":"done"}
```

```sh
$ curl -H "Authorization: Bearer my-token-1" http://localhost:5000/my-action-1
{"id":"my-action-1","status":"failed"}
```

### Make it persistent

To make the service a system-wide daemon, create `/etc/systemd/system/webtrigger.service`

```
[Unit]
Description=Web Trigger service to allow running scripts from CI/CD jobs
After=network.target

[Service]
ExecStart=/usr/local/bin/webtrigger /path/to/config.yaml
# Required on some systems
#WorkingDirectory=/usr/local/bin
Restart=always
# Restart service after 10 seconds if the service crashes
RestartSec=10
# Output to syslog
StandardOutput=syslog
StandardError=syslog
SyslogIdentifier=webtrigger
Type=simple
#User=<alternate user>
#Group=<alternate group>
Environment=

[Install]
WantedBy=multi-user.target
```

- Specify `User` and `Group` to drop `root` privileges

Reload Systemd's config:

```sh
$ sudo systemctl daemon-reload
```

Enable the service:

```sh
$ sudo systemctl enable webtrigger.service
```

Start the service:

```sh
$ sudo systemctl start webtrigger.service
```
<!--
### TLS encryption

On a typical scenario you will want your access tokens to travel encrypted.

If you are running a reverse proxy like Nginx, you can forward incoming HTTPS requests to webtrigger on a local port. But if Nginx itself is running within a Docker container, you might have issues forwarding requests back to webtrigger on the host system.

For such scenarios, you can enable TLS encryption right on webtrigger itself.

Then, pass the `TLS_CERT` and `TLS_KEY` environment variables. 

```sh
$ PORT=1234 TLS_CERT=/path/to/server.cert TLS_KEY=/path/to/server.key node .
Using ./triggers.yaml as the config file
Listening on https://0.0.0.0:1234
```

You can also pass `TLS_CHAIN` to specify the certificate chain of your CA.

```sh
$ PORT=1234 TLS_CERT=/path/to/server.pem TLS_KEY=/path/to/server.pem TLS_CHAIN=/path/to/chain.pem node .
Using ./triggers.yaml as the config file
Listening on https://0.0.0.0:1234
```

#### Self signed

Self signed certificates can also be used:

```sh
$ openssl req -nodes -new -x509 -keyout server.key -out server.cert
# enter any dummy data

$ chmod 400 server.key server.cert
```

Just tell `curl` to ignore the certificate credentials and you are good to go:

```sh
$ curl --insecure -H "Authorization: Bearer my-token-1" -X POST https://my-host:5000/my-action-1
OK
```

-->

## TO DO

- [ ] Prevent blocking 3+ requests while still ongoing
- [x] Handle timeouts
- [ ] Allow TLS certificates
