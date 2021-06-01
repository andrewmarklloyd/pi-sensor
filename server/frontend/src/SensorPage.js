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
              <p>Last health check: Unknown</p>
          </Card.Body>
        </Card>
        <Link to={{pathname: "/"}}><Button color="secondary">Back</Button></Link>
        </Page.Content>
      </SiteWrapper>
    );
  }
}

export default SensorPage;


