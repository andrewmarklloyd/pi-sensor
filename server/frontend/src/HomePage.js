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
    if (window.location.protocol === "https:") {
      socket = socketIOClient.connect(`wss://${window.location.host}/ws`, { transports: ['websocket'] });
    } else {
      socket = socketIOClient.connect(`ws://localhost:8080`, { transports: ['websocket'] });
    }
    socket.on("garage", function(message) {
      console.log("message in home component:", message)
    })
    socket.on("sensor", function(message) {
      console.log("message in home component:", message)
    })
    socket.on("status", function(message) {
      console.log("message in home component:", message)
    })
    socket.on("sensor/status", function(message) {
      console.log("message in home component:", message)
    })
  }

  componentDidMount() {
    fetch('/sensors')
    .then(res => res.json())
    .then(json => {
      console.log(json)
      this.setState({data: json})
    })
    .catch(err => {
      console.log(err)
    })
  }

  render() {
    return (
      <SiteWrapper>
        <Page.Content>
        <Grid.Row cards={true}>
          <Grid.Col sm={6} lg={3}>
            <Sensor key="garage" source="garage" socket={socket}/>
          </Grid.Col>
        </Grid.Row>
        </Page.Content>
      </SiteWrapper>
      // <SiteWrapper>
      //   <Page.Content>
      //   <Grid.Row cards={true}>
      //     <Grid.Col sm={6} lg={3}>
      //       {Object.keys(this.state.data).map(key => (
      //         <Sensor key={key} source={key} socket={socket}/>
      //       ))}
      //     </Grid.Col>
      //   </Grid.Row>
      //   </Page.Content>
      // </SiteWrapper>
    );
  }
}

export default Home;
