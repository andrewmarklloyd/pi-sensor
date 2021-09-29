import React, { Component } from "react";

import {
  Page,
  Card,
  Table
} from "tabler-react";

import { unixToDate } from "./DataModel";

import SiteWrapper from "./SiteWrapper";

class ReportPage extends Component {
  constructor(props) {
    super(props)
    this.state = {messages: []}
  }

  getReport() {
    var component = this
    fetch("/api/report", {
      method: 'GET',
      credentials: 'same-origin',
      headers: {
        'Content-Type': 'application/json'
      },
      referrerPolicy: 'no-referrer'
    })
    .then(r => r.json())
    .then(res => {
      component.state.messages = res.messages
      component.setState(component.state)
    })
  }

  
  render() {
    return (
      <SiteWrapper>
        <Page.Content>
        <Card>
          <Card.Header>
            <button onClick={() => this.getReport()}>
              Submit
            </button>
          </Card.Header>
          <Card.Body>
            <Table>
              <Table.Header>
                <Table.ColHeader>Source</Table.ColHeader>
                <Table.ColHeader>Status</Table.ColHeader>
                <Table.ColHeader>Timestamp</Table.ColHeader>
              </Table.Header>
              <Table.Body>
              {this.state.messages.map(item => (
                <Table.Row>
                  <Table.Col>{item.source}</Table.Col>
                  <Table.Col>{item.status}</Table.Col>
                  <Table.Col>{unixToDate(item.timestamp)}</Table.Col>
                </Table.Row>
              ))
              }
              </Table.Body>
            </Table>
          </Card.Body>
        </Card>
        </Page.Content>
      </SiteWrapper>
    );
  }
}

export default ReportPage;


