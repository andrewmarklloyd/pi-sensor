// @flow

import React, { Component } from "react";
import socketIOClient from "socket.io-client";

import {
  Page,
  Grid,
  Card,
} from "tabler-react";

import SiteWrapper from "./SiteWrapper";
import Sensor from "./Sensor";
import { translateStatus, timeSince } from "./DataModel";

var socket
var socket2

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

    socket2 = new WebSocket(`${url}/ws/`);
    socket2.onopen = () => {
      console.log("connection opened")
      const msg = {
        type: "message",
        channel: "test/message",
        text: "this is from the client",
        date: Date.now(),
      };
      socket2.send(JSON.stringify(msg));
    };
    socket2.onclose = function () {
      console.log("closed connection")
    };

    socket = socketIOClient.connect(`${url}`, { transports: ['websocket'] });
    socket.on("connect", function() {})
  }

  componentDidMount() {
    socket2.onmessage = (event) => {
      console.log("message", event.data)
    }
    var component = this
    socket.on("sensor/list", function(data) {
      var d = JSON.parse(data)
      console.log(d)
      var sensors = []
      if (d.sensors == null) {
        d.sensors = []
      }
      d.sensors.sort(function(a, b) {
        return a.source > b.source ? 1 : -1
      });
      d.sensors.forEach(element => {
        var updated = translateStatus(element.status)
        sensors.push({
          source: element.source,
          status: element.status,
          timestamp: element.timestamp,
          timesince: timeSince(element.timestamp),
          icon: updated.icon,
          color: updated.color,
          armed: d.arming[element.source],
          version: element.version
        })
      })
      component.setState({data: sensors})
    })
  }

  render() {
    return (
      <SiteWrapper>
        <Page.Content>
        {this.state.data.length > 0 ? (
          <Grid.Row cards={true}>
          <Grid.Col sm={6} lg={3}>
            {this.state.data.map(item => (
              <Sensor key={item.source} source={item.source} socket={socket} status={item.status} icon={item.icon} color={item.color} timestamp={item.timestamp} timesince={item.timesince} armed={item.armed} version={item.version}/>
            ))
            }
          </Grid.Col>
        </Grid.Row>
        ) : (
          <Card>
          <Card.Header>
              <Card.Title>No sensors currently connected</Card.Title>
          </Card.Header>
        </Card>
        )}
        </Page.Content>
      </SiteWrapper>
    );
  }
}

export default Home;
