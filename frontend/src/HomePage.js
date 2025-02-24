// @flow

import React, { Component } from "react";
import Grid2 from '@mui/material/Grid2';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
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

    socket = new WebSocket(`${url}/ws/`);
  }

  componentDidMount() {
    var component = this
    socket.addEventListener("message", function(event) {
      var data = JSON.parse(event.data)
      if (data.channel === "sensor/list") {
        var d = JSON.parse(data.message)
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
        }
    })
  }

  render() {
    return (
      <div>
        <Grid2 container spacing={2}>
        {this.state.data.length > 0 ? (
          <div>
            {this.state.data.map(item => (
              <Sensor key={item.source} source={item.source} socket={socket} status={item.status} icon={item.icon} color={item.color} timestamp={item.timestamp} timesince={item.timesince} armed={item.armed} version={item.version}/>
            ))
            }
          </div>
            
        ) : (
        <Card>
          <CardContent>No sensors currently connected</CardContent>
        </Card>
        )
        }
        </Grid2>
      </div>
      
    );
  }
}

export default Home;
