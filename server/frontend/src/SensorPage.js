import React, { Component } from "react";
import { Link } from "react-router-dom";

import {
  Page,
  Card,
  Button,
} from "tabler-react";

import SiteWrapper from "./SiteWrapper";

class SensorPage extends Component {
  constructor(props) {
    super(props)
    this.state = this.props.location.state
  }

  restartSensor(source) {
    if (window.confirm('Are you sure you wish to restart the sensor?')) {
      fetch("/api/sensor/restart", {
        method: 'POST',
        credentials: 'same-origin',
        headers: {
          'Content-Type': 'application/json'
        },
        referrerPolicy: 'no-referrer',
        body: JSON.stringify({source: source})
      })
      .then(r => r.json())
    }
  }

  render() {
    return (
      <SiteWrapper>
        <Page.Content>
        <Card>
          <Card.Header>
              <Card.Title>Sensor: {this.state.source}</Card.Title>
          </Card.Header>
          <Card.Body>
              <p>Last activity: {this.state.timesince}</p>
              <button onClick={() => this.restartSensor(this.state.source)}>
                Restart
              </button>
          </Card.Body>
        </Card>
        <Link to={{pathname: "/"}}><Button color="secondary">Back</Button></Link>
        </Page.Content>
      </SiteWrapper>
    );
  }
}

export default SensorPage;


