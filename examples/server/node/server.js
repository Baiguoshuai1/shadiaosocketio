const { Server } = require("socket.io");
const customParser = require('socket.io-msgpack-parser');

const io = new Server(2233, {
    // parser: customParser // enable binary msg, and you need to set `BinaryMessage` to true for client
});


io.on("connection", (socket) => {
    logWithTimestamp('on connection, id:', socket.id)

    socket.on('message', function(arg1, arg2, arg3) {
        logWithTimestamp('on message:', arg1, arg2, arg3)
    });

    socket.on('/ackFromClient', function(arg1, arg2, func) {
        logWithTimestamp('on ackFromClient:', arg1, arg2)
        // sleep 3s
        setTimeout(() => func(3, { text: 'respFromServer' }, "string"), 3000)
    });

    socket.emit('message', { id: 1, channel: "{\"chinese\":\"中文才是最屌的\"}"})

    logWithTimestamp('send method: ackFromServer, wait for 2s')
    socket.emit('/ackFromServer', "server", 2, function(arg1, arg2) {
        logWithTimestamp('on ackFromServer cb:', arg1, arg2)
    });
});

logWithTimestamp("running...")

function logWithTimestamp(...args) {
    const timestamp = new Date();
    const hours = timestamp.getHours().toString().padStart(2, '0');
    const minutes = timestamp.getMinutes().toString().padStart(2, '0');
    const seconds = timestamp.getSeconds().toString().padStart(2, '0');
    const milliseconds = timestamp.getMilliseconds().toString().padStart(3, '0');

    console.log(`${hours}:${minutes}:${seconds}.${milliseconds}`, '[server]', ...args);
}
