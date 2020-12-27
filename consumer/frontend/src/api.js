var ws
if (window.location.protocol === "https:") {
  ws = new WebSocket(`wss://${window.location.origin}/ws`);
} else {
  ws = new WebSocket(`ws://localhost:8080/ws`);
}

function setupWebSocket(){
  ws.onclose = function(){
    console.log("Closed connection")
    // setTimeout(setupWebSocket, 1000);
  }

  ws.onopen = function(evt) {
    console.log("Opened connection")
  }

  ws.onerror = function(evt) {
    console.log("Websocket error: " + evt.data);
  }
}

function subscribeToChange(cb) {
  ws.onmessage = function(evt) {
    var data = JSON.parse(evt.data)
    console.log(data.state, data.source)
    cb(data)
  }
}

export { setupWebSocket, subscribeToChange };
