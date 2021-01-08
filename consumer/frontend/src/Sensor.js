import React, { Component } from 'react';

import {
  StampCard,
} from "tabler-react";

class Sensor extends Component {
  constructor(props) {
    super(props)
    console.log("Sensor.js source:", this.props.source, this.props.socket)
    this.props.socket.on("Office Door", function(message) {
      console.log("hard coded source")
    })
    this.props.socket.on(this.props.source, function(message) {
      console.log(this.props.source, " has a new message:", message);
      //   try {
  //     // var data = JSON.parse(evt.data)
  //     // console.log(data)
  //     // var state = data.state
  //     // component.setState({
  //     //   color: state === "OPEN" ? "red" : "green",
  //     //   source: data.source,
  //     //   icon: state === "OPEN" ? "unlock" : "lock",
  //     //   timestamp: "10 min ago"
  //     // })
  //   } catch(e) {
  //     console.log("Error parsing json:", e)
  //   }
    })
  }

  state = {
    color: "grey",
    source: this.props.source,
    icon: "zap-off",
    timestamp: "Unknown"
  };

  render() {
    return (
      <StampCard
        color={this.state.color}
        icon={this.props.icon || this.state.icon}
        header={
          <a href="/">
            {this.props.source}
          </a>
        }
        footer={this.props.timestamp || this.state.timestamp}
      />
    );
  }
}

export default Sensor;
