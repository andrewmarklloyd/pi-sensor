import React, { Component } from "react";
import { DataGrid } from '@mui/x-data-grid';
import Card from '@mui/material/Card';
import FormControl from '@mui/material/FormControl';

import {
  Page,
  Card as CardTabler,
  Form,
  Table
} from "tabler-react";

import { trimVersion, unixToDate } from "./DataModel";

const columns = [
  {
    field: 'timestamp',
    renderHeader: () => (<strong>{'Time'}</strong>),
    // width: 110,
    editable: false,
  },
  {
    field: 'source',
    renderHeader: () => (<strong>{'Door'}</strong>),
    // width: 210,
    editable: false,
  },
  {
    field: 'status',
    renderHeader: () => (<strong>{'Status'}</strong>),
    // width: 150,
    editable: false
  },
  {
    field: 'version',
    renderHeader: () => (<strong>{'Version'}</strong>),
    // width: 100,
    editable: false
  }
]

class ReportPage extends Component {
  constructor(props) {
    super(props)
    this.state = {rows: [], messages: [], sensors: ['', 'All'], numPages: 1, page: 1}
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
      res.sensors.map((item, index) => (
        component.state.sensors.push(item)
      ))
      component.setState(component.state)
    })
  }

  handleChange(e) {
    var value = e.target.value
    var sensor
    var page
    var component = this
    if (value === '') {
      this.setState({messages: []})
      return
    }
    if (isNaN(value)) {
      sensor = value
      page = sensor === this.state.sensor ? this.state.page : 1
    } else {
      page = value
      sensor = this.state.sensor
    }
    component.setState({page: page, sensor: sensor})
    fetch(`/api/report?sensor=${sensor}&page=${page}`, {
      method: 'GET',
      credentials: 'same-origin',
      headers: {
        'Content-Type': 'application/json'
      },
      referrerPolicy: 'no-referrer',
    })
    .then(r => r.json())
    .then(res => {
      let messages = []
      res.messages.map(item => {
        messages.push({
          timestamp: unixToDate(item.timestamp),
          source: item.source,
          status: item.status,
          version: trimVersion(item.version)
        })
      })
      component.setState({rows: messages, messages: messages, numPages: res.numPages})
    })
  }

  getPageOptions() {
    var arr = []
    for (var i = 1; i <= this.state.numPages; i++) {
      arr.push(i)
    }
    return (arr.map((item, index) => (
      <option key={index} value={item}>
        {item}
      </option>
    )))
  }
  
  render() {
    return (
      <div>
        <Card>
        <FormControl fullWidth>
          
        </FormControl>
        </Card>
        <Form.Group label="">
          <h4>Sensor</h4>
          <Form.Select onChange={this.handleChange}>
            {this.state.sensors.map((item, index) => (
              <option key={index} value={item}>
                {item}
              </option>
              ))}
          </Form.Select>
          <Form.Select onChange={this.handleChange}>
            {this.getPageOptions()}
          </Form.Select>
        </Form.Group>
        <Card sx={{ m: 10}}>
        <DataGrid
        sx={{
          '.MuiDataGrid-columnHeaderTitleContainer': {
            whiteSpace: 'normal',
            wordWrap: 'break-word',
            padding: '5px'
          },
          '.MuiDataGrid-cell': {
              whiteSpace: 'normal',
              wordWrap: 'break-word',
              padding:'5px'
          }
        }}
        getRowHeight={() =>{ return 'auto'}}
        getRowId={(row) => row.timestamp}
        rows={this.state.rows}
        columns={columns}
        initialState={{
          pagination: {
              paginationModel: {
              pageSize: 50,
              },
          },
        }}
        pageSizeOptions={[25, 50, 100]}
      />
        </Card>

        {/* <Page.Content>
        <CardTabler>
          <CardTabler.Header>
            <Form.Group label="">
              <h4>Sensor</h4>
              <Form.Select onChange={this.handleChange}>
                {this.state.sensors.map((item, index) => (
                  <option key={index} value={item}>
                    {item}
                  </option>
                  ))}
              </Form.Select>
              <Form.Select onChange={this.handleChange}>
                {this.getPageOptions()}
              </Form.Select>
            </Form.Group>
          </CardTabler.Header>
          <CardTabler.Body>
            <Table>
              <Table.Header>
                <Table.ColHeader>Time</Table.ColHeader>
                <Table.ColHeader>Door</Table.ColHeader>
                <Table.ColHeader>Status</Table.ColHeader>
                <Table.ColHeader>Version</Table.ColHeader>
              </Table.Header>
              <Table.Body>
              {this.state.messages.map(item => (
                <Table.Row>
                  <Table.Col>{unixToDate(item.timestamp)}</Table.Col>
                  <Table.Col>{item.source}</Table.Col>
                  <Table.Col>{item.status}</Table.Col>
                  <Table.Col>{trimVersion(item.version)}</Table.Col>
                </Table.Row>
              ))
              }
              </Table.Body>
            </Table>
          </CardTabler.Body>
        </CardTabler>
        </Page.Content> */}
      </div>
    );
  }
}

export default ReportPage;


