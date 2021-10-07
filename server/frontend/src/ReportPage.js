import React, { Component } from "react";

import {
  Page,
  Card,
  Form,
  Table
} from "tabler-react";

import { unixToDate } from "./DataModel";

import SiteWrapper from "./SiteWrapper";

class ReportPage extends Component {
  constructor(props) {
    super(props)
    this.state = {messages: [], sensors: ['', 'All']}
    this.handleChange = this.handleChange.bind(this);
  }

  componentDidMount() {
    var component = this
    fetch("/api/sensor/all", {
      method: 'GET',
      credentials: 'same-origin',
      headers: {
        'Content-Type': 'application/json'
      },
      referrerPolicy: 'no-referrer'
    })
    .then(r => r.json())
    .then(res => {
      res.sensors.map((item, index) => {
        component.state.sensors.push(item)
      })
      component.setState(component.state)
    })
  }

  handleChange(e) {
    var value = e.target.value
    if (value === '') {
      this.setState({messages: []})
      return
    }
    var component = this
    fetch(`/api/report?sensor=${value}`, {
      method: 'GET',
      credentials: 'same-origin',
      headers: {
        'Content-Type': 'application/json'
      },
      referrerPolicy: 'no-referrer',
    })
    .then(r => r.json())
    .then(res => {
      component.setState({messages: res.messages})
    })
  }
  
  render() {
    return (
      <SiteWrapper>
        <Page.Content>
        <Card>
          <Card.Header>
            <Form.Group label="">
              <h4>Sensor</h4>
              <Form.Select onChange={this.handleChange}>
                {this.state.sensors.map((item, index) => (
                  <option key={index} value={item}>
                    {item}
                  </option>
                  ))}
              </Form.Select>
            </Form.Group>
          </Card.Header>
          <Card.Body>
            <Table>
              <Table.Header>
                <Table.ColHeader>Time</Table.ColHeader>
                <Table.ColHeader>Door</Table.ColHeader>
                <Table.ColHeader>Status</Table.ColHeader>
              </Table.Header>
              <Table.Body>
              {this.state.messages.map(item => (
                <Table.Row>
                  <Table.Col>{unixToDate(item.timestamp)}</Table.Col>
                  <Table.Col>{item.source}</Table.Col>
                  <Table.Col>{item.status}</Table.Col>
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


