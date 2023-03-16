const io = require("socket.io-client")
const socket = io("ws://127.0.0.1:2233",{transports: ['websocket']})

// listen for messages
socket.on('message', function(msg) {
    console.log('[client] received msg:', msg);
});

// sending ack response
socket.on('/ackFromServer', function(a, b, f) {
    console.log('[client] received ack:', a , b);
    f({ id: 5, channel: 'js channel' }, 6)
});

socket.on('/admin', function(msg) {
    console.log('[client] received admin msg:', msg);
});

socket.on('connect', function () {
    console.log('[client] socket connected');

    socket.emit('message', "js", { id: 1, channel: "js" }, 2);

    // ack
    socket.emit('/ackFromClient',  { id: 3, channel: "js ack" }, 4, (a, b, c) => {
        console.log('[client] ack cb:', a, b, c)
    });
});

socket.on('disconnect', function (e) {
    console.log('[client] socket disconnect', e);
});

socket.on('connect_error', function (e) {
    console.log('[client] connect_error', e)
});
