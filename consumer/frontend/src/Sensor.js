import React, { Component } from 'react';

import {
  StampCard,
} from "tabler-react";


var ws

class Sensor extends Component {
  constructor(props) {
    super(props);
    setupSockets(this)
  }

  state = {
    color: "red",
    source: "",
    icon: "unlock",
    timestamp: "10 min ago"
  };

  render() {
    return (
      <StampCard
        color={this.state.color}
        icon={this.state.icon}
        header={
          <a href="/">
            {this.state.source}
          </a>
        }
        footer={this.state.timestamp}
      />
    );
  }
}

function setupSockets(sensorComponent) {
  if (window.location.protocol === "https:") {
    ws = new WebSocket(`wss://${window.location.host}/ws`);
  } else {
    ws = new WebSocket(`ws://localhost:8080/ws`);
  }
  var component = sensorComponent
  ws.onerror = function(evt) {
    console.log("Websocket error: " + evt.data);
  }

  ws.onmessage = function(evt) {
    console.log(evt.data)
    try {
      var data = JSON.parse(evt.data)
      var state = data.state
      component.setState({
        color: state === "OPEN" ? "red" : "green",
        source: data.source,
        icon: state === "OPEN" ? "unlock" : "lock",
        timestamp: "10 min ago"
      })
    } catch(e) {
      console.log("Error parsing json:", e)
    }
  }
  
  ws.onopen = function(evt) {
    console.log("Connected")
  }
  
  ws.onclose = function(){
    setTimeout(() => {
      setupSockets(component)
    }, 5000)
    console.log("Disconnected, attempting reconnect in 5 seconds.")
  }
}

export default Sensor;
