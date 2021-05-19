// @flow

import React, { Component } from "react";
import socketIOClient from "socket.io-client";

import {
  Page,
  Grid,
} from "tabler-react";

import SiteWrapper from "./SiteWrapper";
import Sensor from "./Sensor";
import { translateStatus, timeSince } from "./DataModel";

var socket

class Home extends Component {
  constructor(props) {
    super(props)
    this.state = { data: [] }
    var url
    if (window.location.protocol === "https:") {
      url = `wss://${window.location.host}`
    } else {
      url = "ws://localhost:8080"
    }
    socket = socketIOClient.connect(`${url}`, { transports: ['websocket'] });
    socket.on("connect", function() {})
  }

  componentDidMount() {
    var component = this
    socket.on("sensor/list", function(data) {
      var d = JSON.parse(data)
      var sensors = []
      d.data.forEach(element => {
        sensors.push({
          source: element.source,
          status: element.status,
          timestamp: element.timestamp,
          timesince: timeSince(element.timestamp),
          icon: "zap-off"
        })
      })
      component.setState({data: sensors})
    })
  }

  render() {
    var state
    return (
      <SiteWrapper>
        <Page.Content>
        <Grid.Row cards={true}>
          <Grid.Col sm={6} lg={3}>
            {this.state.data.map(item => (
              state = translateStatus(item.status), // eslint-disable-line no-sequences
              <Sensor key={item.source} source={item.source} socket={socket} status={item.status} icon={state.icon} color={state.color} timestamp={item.timestamp} timesince={item.timesince}/>
            ))
            }
          </Grid.Col>
        </Grid.Row>
        </Page.Content>
      </SiteWrapper>
    );
  }
}

export default Home;
