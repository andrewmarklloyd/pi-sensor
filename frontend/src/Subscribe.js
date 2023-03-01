import React, { Component } from "react";

import {
  Page,
} from "tabler-react";

import SiteWrapper from "./SiteWrapper";

class SubscribePage extends Component {
  constructor(props) {
    super(props)
  }

  componentDidMount() {
    
  }

  
  render() {
    return (
      <SiteWrapper>
        <Page.Content>
        <div>Subscription to push notifications coming soon!</div>
        </Page.Content>
      </SiteWrapper>
    );
  }
}

export default SubscribePage;
