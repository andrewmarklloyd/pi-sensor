import React, { Component } from "react";

import {
  Page,
  Card,
} from "tabler-react";

import SiteWrapper from "./SiteWrapper";

class SensorPage extends Component {
  render() {
    return (
      <SiteWrapper>
        <Page.Content>
        <Card>
          <Card.Header>
              <Card.Title>Sensor: {this.props.location.state.source}</Card.Title>
          </Card.Header>
          <Card.Body>
              <p>Last activity: 2 min ago</p>
              <p>Last health check: 10 sec ago</p>
          </Card.Body>
          <Card.Footer>Card footer</Card.Footer>
        </Card>
        </Page.Content>
      </SiteWrapper>
    );
  }
}

export default SensorPage;
