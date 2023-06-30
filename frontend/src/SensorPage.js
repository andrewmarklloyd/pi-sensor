import React, { Component } from "react";
import { Link } from "react-router-dom";
import { trimVersion, unixToDate } from "./DataModel";


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
    var component = this
    fetch("/api/sensor/getOpenTimeout?source="+component.state.source, {
      method: 'GET',
      credentials: 'same-origin',
      headers: {
        'Content-Type': 'application/json'
      },
      referrerPolicy: 'no-referrer'
    })
    .then(r => r.json())
    .then(res => {
      if (res.status === "success") {
        component.state.openTimeout = res.openTimeout
        component.setState(component.state)
      } else {
        console.log("error getting openTimeout: ", res.error)
      }
    })
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

  toggleArm(source) {
    var component = this
    if (window.confirm('Are you sure you wish to toggle arm/disarm?')) {
      fetch("/api/sensor/arming", {
        method: 'POST',
        credentials: 'same-origin',
        headers: {
          'Content-Type': 'application/json'
        },
        referrerPolicy: 'no-referrer',
        body: JSON.stringify({source: source})
      })
      .then(r => r.json())
      .then(res => {
        component.state.armed = res.armed
        component.setState(component.state)
      })
    }
  }

  handleChange(a) {
    this.setState({openTimeout: parseInt(a.target.value)})
  }

  submitOpenTimeout(event) {
    event.preventDefault();
    fetch("/api/sensor/openTimeout", {
      method: 'POST',
      credentials: 'same-origin',
      headers: {
        'Content-Type': 'application/json'
      },
      referrerPolicy: 'no-referrer',
      body: JSON.stringify({source: this.state.source, openTimeout: this.state.openTimeout})
    })
    .then(r => r.json())
    .then(res => {
      if (res.status === "success") {
        alert("Successfully updated open timeout")
      } else {
        alert("Error updating open timeout: " + res.error)
      }
    })
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
              <p>Last activity: {unixToDate(this.state.timestamp)}</p>
              <p>Alerting: {this.state.armed === "true" ? "Armed" : "Disarmed"}</p>
              <p>Version: {trimVersion(this.state.version)}</p>
              <div>
                <button onClick={() => this.restartSensor(this.state.source)}>
                  Restart
                </button>
              </div>
              <div>
                <button onClick={() => this.toggleArm(this.state.source)}>
                  Arm/Disarm
                </button>
              </div>
              <div>
                <form onSubmit={this.submitOpenTimeout.bind(this)}>
                  <label>
                    Name:
                    <input type="number" min="1" max="60" value={this.state.openTimeout} onChange={this.handleChange.bind(this)} />
                  </label>
                  <input type="submit" value="Submit" />
                </form>
              </div>
          </Card.Body>
        </Card>
        <Link to={{pathname: "/"}}><Button color="secondary">Back</Button></Link>
        </Page.Content>
      </SiteWrapper>
    );
  }
}

export default SensorPage;


