// @flow

import React, { Component } from "react";

import {
  Page,
  Grid,
} from "tabler-react";

import SiteWrapper from "./SiteWrapper";
import Sensor from "./Sensor";

class Home extends Component {
  constructor(props) {
    super(props)
    this.state = { data: [] }
  }

  componentDidMount() {
    fetch('http://localhost:8080/sensors')
    .then(res => res.json())
    .then(json => {
      this.setState({data: json})
    })
    .catch(err => {
      console.log(err)
    })
  }

  render() {
    return (
      <SiteWrapper>
        <Page.Content title="Dashboard">
        <Grid.Row cards={true}>
          <Grid.Col sm={6} lg={3}>
            {Object.keys(this.state.data).map(key => (
              <Sensor name={key}/>
            ))}
          </Grid.Col>
        </Grid.Row>
        </Page.Content>
      </SiteWrapper>
    );
  }
}

export default Home;
