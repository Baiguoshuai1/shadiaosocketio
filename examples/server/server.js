const { Server } = require("socket.io");
const customParser = require('socket.io-msgpack-parser');

const io = new Server(2233, {
    // parser: customParser
});

io.on("connection", (socket) => {
    // ...
    socket.on('message', function(arg1, arg2, arg3) {
       console.log('[server] received message:', arg1, arg2, arg3)
    });

    // listen ack event
    socket.on('/ackFromClient', function(arg1, arg2, func) {
        console.log('[server] received ack:', arg1, arg2)
        func(1, { text: 'resp' }, "server")
    });

    socket.emit('message', { id: 2, channel: "{\"chinese\":\"中文才是最屌的\"}"})

    // emit ack event
    socket.emit('/ackFromServer', "vue", 3, function(arg1, arg2) {
        console.log('[server] ack cb:', arg1, arg2)
    });
});

console.log("[server] starting server...")