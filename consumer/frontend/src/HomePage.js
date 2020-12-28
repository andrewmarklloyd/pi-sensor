// @flow

import * as React from "react";

import {
  Page,
  Grid,
  StampCard,
} from "tabler-react";

import SiteWrapper from "./SiteWrapper";
import Sensor from "./Sensor";

function Home() {
  return (
    <SiteWrapper>
      <Page.Content title="Dashboard">
      <Grid.Row cards={true}>
        <Grid.Col sm={6} lg={3}>
          <Sensor/>
        </Grid.Col>
        <Grid.Col sm={6} lg={3}>
          <StampCard
            color="secondary"
            icon="zap-off"
            header={
              <a href="#">
                Garage Door
              </a>
            }
            footer={"timestamp"}
          />
        </Grid.Col>
        <Grid.Col sm={6} lg={3}>
          <StampCard
            color="green"
            icon="lock"
            header={
              <a href="#">
                Front Door
              </a>
            }
            footer={"timestamp"}
          />
        </Grid.Col>
      </Grid.Row>
      </Page.Content>
    </SiteWrapper>
  );
}

export default Home;
