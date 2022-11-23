const io = require("socket.io-client")
const socket = io("ws://127.0.0.1:2233",{transports: ['websocket']})

// listen for messages
socket.on('message', function(msg) {
    console.log('received msg:', msg);
});

socket.on('/admin', function(msg) {
    console.log('received admin msg:', msg);
});

socket.on('connect', function () {
    console.log('socket connected');

    socket.emit('message', "1", { id: 2, text: "js" }, 2);
});
socket.on('connect_error', function (e) {
    console.log('connect_error', e)
});
