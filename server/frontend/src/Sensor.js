import React, { Component } from 'react';
import translateStatus from "./DataModel";

import {
  StampCard,
} from "tabler-react";

class Sensor extends Component {
  constructor(props) {
    super(props)
    var source = this.state.source
    var component = this
    this.props.socket.on("sensor/status", function(data) {
      if (data.source == source) {
        var state = translateStatus(data.status)
        component.setState({
          color: state.status,
          source: data.source,
          icon: state.status,
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
        color={this.props.color || this.state.color}
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
