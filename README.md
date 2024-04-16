golang socket.io
================

GoLang implementation of [socket.io](http://socket.io) library, client and server.

Examples directory contains simple clients and servers.

### Get It

```sh
go get -u github.com/Baiguoshuai1/shadiaosocketio
```

### Debug
```sh
DEBUG=1 go run client.go
```

### Simple Clients Usage
[client.html](./examples/client/html/client.html)
or
[client.go](./examples/client/client.go)

### Simple Servers Usage
[server.js](./examples/server/node/server.js)
or
[server.go](./examples/server/server.go)


### Compatibility
<table style="text-align: center">
    <tr style="font-weight:bold">
        <td></td>
        <td></td>
        <td colspan="9">Client</td>
    <tr>
    <tr style="font-weight:bold">
        <td rowspan="3"></td>
        <td></td>
        <td colspan="2">JavaScript（browser）</td>
        <td colspan="2">JavaScript（Node.js）</td>
        <td colspan="2">Golang（client）</td>
    <tr>
    <tr>
        <td></td>
        <td>v3.x</td>
        <td>v4.x</td>
        <td>v3.x</td>
        <td>v4.x</td>
        <td>-</td>
    <tr>
    <tr style="font-weight:bold">
        <td style="font-weight:bold">Golang（server）</td>
        <td></td>
        <td>✅</td>
        <td>✅</td>
        <td>✅</td>
        <td>✅</td>
        <td>✅</td>
    <tr>
    <tr>
        <td rowspan="7" style="font-weight:bold">Node.js（server）</td>
        <td>v3.x</td>
        <td>✅</td>
        <td>✅</td>
        <td>✅</td>
        <td>✅</td>
        <td>✅</td>
    <tr>
    <tr>
        <td>v4.x</td>
        <td>✅</td>
        <td>✅</td>
        <td>✅</td>
        <td>✅</td>
        <td>✅</td>
    <tr>
</table>




