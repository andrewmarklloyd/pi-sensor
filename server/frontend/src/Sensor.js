import React, { Component } from 'react';

import {
  StampCard,
} from "tabler-react";

class Sensor extends Component {
  constructor(props) {
    super(props)
    var source = this.state.source
    var component = this
    this.props.socket.on("sensor/status", function(data) {
      console.log("sensor/status:", data)
      if (data.source == source) {
        component.setState({
          color: data.status === "OPEN" ? "red" : "green",
          source: data.source,
          icon: data.status === "OPEN" ? "unlock" : "lock",
          timestamp: "10 min ago"
        })  
      }
    })
  }

  componentDidMount() {
    
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
