var ws
if (window.location.protocol === "https:") {
  ws = new WebSocket(`wss://${window.location.origin}/ws`);
} else {
  ws = new WebSocket(`ws://localhost:8080/ws`);
}
// var ws = new WebSocket(`ws://${window.location.host}/ws`);

function setupWebSocket(){
  console.log("setting up websocket")
  // if (location.protocol == "https:") {
  //   ws = new WebSocket(`wss://${location.host}/ws`);
  // } else {
  //   ws = new WebSocket(`ws://${location.host}/ws`);
  // }

  // var ws = new WebSocket(`ws://localhost:8080/ws`);
  ws.onclose = function(){
    console.log("Closed connection")
    // setTimeout(setupWebSocket, 1000);
  }

  ws.onopen = function(evt) {
    console.log("Opened connection")
  }

  // ws.onmessage = function(evt) {
  //   var data = JSON.parse(evt.data)
  //   console.log(data.state, data.source)
  // }

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
