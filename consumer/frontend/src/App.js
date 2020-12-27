// import { setupWebSocket, subscribeToMessages, subscribeToConnect, subscribeToDisconnect } from './websocket';
import React, { Component } from 'react';
import './App.css';

var ws

class App extends Component {
  constructor(props) {
    super(props);
    
    var app = this
    setupSockets(app)
  }

  state = {
    data: 'no data yet'
  };

  render() {
    return (
      <div className="App">
        <p className="App-intro">
        Door status: {this.state.data.state}
        </p>
      </div>
    );
  }
}

function setupSockets(app) {
  if (window.location.protocol === "https:") {
    ws = new WebSocket(`wss://${window.location.origin}/ws`);
  } else {
    ws = new WebSocket(`ws://localhost:8080/ws`);
  }

  ws.onerror = function(evt) {
    console.log("Websocket error: " + evt.data);
  }

  ws.onmessage = function(evt) {
    var data = JSON.parse(evt.data)
    console.log(data.state, data.source)
    app.setState({
      data
    })
  }
  
  ws.onopen = function(evt) {
    console.log("Connected")
  }
  
  ws.onclose = function(){
    setTimeout(() => {
      setupSockets(app)
    }, 5000)
    console.log("Disconnected, attempting reconnect in 5 seconds.")
  }
}

export default App;
