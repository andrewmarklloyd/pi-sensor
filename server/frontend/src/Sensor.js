import { React, Component } from 'react';
import { Link } from "react-router-dom";
import { translateStatus, timeSince } from "./DataModel";

import {
  StampCard,
} from "tabler-react";

class Sensor extends Component {
  constructor(props) {
    super(props)
    var source = this.state.source
    var component = this
    this.props.socket.on("sensor/status", function(data) {
      if (data.source === source) {
        var updated = translateStatus(data.status)
        component.setState({
          color: updated.color,
          source: data.source,
          icon: updated.icon,
          timestamp: data.timestamp,
          timesince: timeSince(data.timestamp)
        })
      }
    })
  }

  state = {
    color: "",
    source: this.props.source,
    icon: "zap-off",
    timestamp: "",
    timesince: ""
  };

  componentDidMount() {
    var component = this
    setInterval(() => {
      component.setState({
        color: component.state.color !== "" ? component.state.color : component.props.color,
        source: component.state.source !== "" ? component.state.source : component.props.source,
        icon: component.state.icon !== "" ? component.state.icon : component.props.icon,
        timestamp: component.state.timestamp !== "" ? component.state.timestamp : component.props.timestamp,
        timesince: timeSince(component.state.timestamp !== "" ? component.state.timestamp : component.props.timestamp)
      })
    }, 60000)
  }

  render() {
    return (
      <StampCard
        color={this.state.color !== "" ? this.state.color : this.props.color}
        icon={this.state.icon !== "zap-off" ? this.state.icon : this.props.icon}
        header={
          <Link
          to={{
            pathname: "/sensor",
            state: {
              source: this.props.source
            }
          }}>
            {this.props.source}
          </Link>
        }
        footer={this.state.timesince !== "" ? this.state.timesince : this.props.timesince}
      />
    );
  }
}

export default Sensor;
