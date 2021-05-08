// @flow

import React, { Component } from "react";
import socketIOClient from "socket.io-client";

import {
  Page,
  Grid,
} from "tabler-react";

import SiteWrapper from "./SiteWrapper";
import Sensor from "./Sensor";

var socket

class Home extends Component {
  constructor(props) {
    super(props)
    this.state = { data: [] }
    var component = this
    var url
    if (window.location.protocol === "https:") {
      url = `wss://${window.location.host}`
    } else {
      url = "ws://localhost:8080"
    }
    socket = socketIOClient.connect(`${url}`, { transports: ['websocket'] });
    socket.on("connect", function() {
      console.log("connected")
    })
    socket.on("sensor/list", function(data) {
      component.setState(JSON.parse(data))
    })
  }

  render() {
    return (
      <SiteWrapper>
        <Page.Content>
        <Grid.Row cards={true}>
          <Grid.Col sm={6} lg={3}>
            {this.state.data.map(item => (
              <Sensor key={item.source} source={item.source} socket={socket}/>
            ))}
          </Grid.Col>
        </Grid.Row>
        </Page.Content>
      </SiteWrapper>
    );
  }
}

export default Home;
