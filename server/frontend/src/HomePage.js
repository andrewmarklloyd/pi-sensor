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
    var url
    if (window.location.protocol === "https:") {
      url = `wss://${window.location.host}/ws`
    } else {
      url = "ws://localhost:8080"
    }
    socket = socketIOClient.connect(`${url}`, { transports: ['websocket'] });
    socket.on("connect", function() {
      console.log("connected")
    })
  }

  componentDidMount() {
    // fetch('/sensors')
    // .then(res => res.json())
    // .then(json => {
    //   this.setState({data: json})
    // })
    // .catch(err => {
    //   console.log(err)
    // })
  }

  render() {
    return (
      <SiteWrapper>
        <Page.Content>
        <Grid.Row cards={true}>
          <Grid.Col sm={6} lg={3}>
            <Sensor source="garage" socket={socket}/>
          </Grid.Col>
          <Grid.Col sm={6} lg={3}>
            <Sensor source="front-door" socket={socket}/>
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
