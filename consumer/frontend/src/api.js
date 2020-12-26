function setupWebSocket(){
  console.log("setting up websocket")
  // if (location.protocol == "https:") {
  //   this.ws = new WebSocket(`wss://${location.host}/ws`);
  // } else {
  //   this.ws = new WebSocket(`ws://${location.host}/ws`);
  // }

  var ws = new WebSocket(`ws://localhost:8080/ws`);
  // this.ws.onclose = function(){
  //   setTimeout(setupWebSocket, 1000);
  // }
  ws.onopen = function(evt) {
    console.log("opened")
  }
}

export { setupWebSocket };
